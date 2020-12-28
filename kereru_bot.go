package main

import "log"
import "fmt"
import "time"
import "github.com/kereru-io/twitter"
import "github.com/kereru-io/twitter/api"
import "database/sql"
import _ "github.com/go-sql-driver/mysql"
import "sync"

var db *sql.DB
var err error
var twitterClient *api.TwitterClient

func databaseSetup() {
	// Create an sql.DB and check for errors
	db, err = sql.Open("mysql", fmt.Sprintf("%v:%v@/%v?charset=utf8mb4", config.DatabaseUser, config.DatabasePassword, config.DatabaseName))
	if err != nil {
		panic(err.Error())
	}

	currentDBVersion := 0
	for i := 0; i < 5; i++ {
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS Migration (ID INTEGER PRIMARY KEY, Version INTEGER)")
		err = db.QueryRow("SELECT Version from Migration ORDER BY Version DESC limit 1").Scan(&currentDBVersion)
		if currentDBVersion == requiredDBVersion {
			break
		}
		log.Print("Waiting on Database upgrade...", err)
		time.Sleep(5 * time.Second)
	}
	if currentDBVersion < requiredDBVersion {
		panic("Database needs upgrading, run kereru app first.")
	}
	if currentDBVersion > requiredDBVersion {
		panic("Wrong Application version!")
	}
}

func twitterSetup() {
	twitterClient = api.NewTwitterClient(config.OauthConsumerKey, config.OauthConsumerSecret, config.OauthToken, config.OauthTokenSecret)
	twitterClient.AccountVerifyCredentials()
}

func markTweetAsError(TweetID string) {
	_, err = db.Exec("UPDATE Tweets SET Status=32 WHERE ID=?", TweetID)
	if err != nil {
		log.Print("Datebase error: ", err)
	}
	tweetAuditEvent(0, TweetID, TWEETERRORED)

}

func markTweetAsSent(TweetID string) {
	_, err = db.Exec("UPDATE Tweets SET Status=8 WHERE ID=?", TweetID)
	if err != nil {
		log.Print("Database error: ", err)
	}
	tweetAuditEvent(0, TweetID, TWEETSENT)
}

func getTweetToSend() string {
	TweetID := "0"
	TimeNow := time.Now().Unix() - config.Delay
	TimeOld := TimeNow - 1740 // 29 minutes ago
	err = db.QueryRow("SELECT ID FROM Tweets WHERE (SendTime>?) and (SendTime<?) AND (Status=4) ORDER BY SendTime ASC", TimeOld, TimeNow).Scan(&TweetID)
	return TweetID
}

func getVideoToUpload() int {
	VideoID := 0
	TimeNow := time.Now().Unix() - config.Delay
	TimeSoon := TimeNow + 1800 // next half hour
	TimeOld := TimeNow - 1800  // last half hour
	err = db.QueryRow("SELECT Videos.ID from Videos INNER JOIN Tweets ON Tweets.VideoA=Videos.ID WHERE Tweets.SendTime>? AND Tweets.SendTime<? AND Videos.MediaTime<?", TimeOld, TimeSoon, TimeOld).Scan(&VideoID)
	return VideoID
}

func getImageToUpload() int {
	ImageID := 0
	TimeNow := time.Now().Unix() - config.Delay
	TimeSoon := TimeNow + 1800 // next half hour
	TimeOld := TimeNow - 1800  // last half hour
	err = db.QueryRow("SELECT Images.ID from Images INNER JOIN Tweets ON Tweets.ImageA=Images.ID WHERE Tweets.SendTime>? AND Tweets.SendTime<? AND Images.MediaTime<?", TimeOld, TimeSoon, TimeOld).Scan(&ImageID)
	return ImageID
}

func getVideoFileName(VideoID int) string {
	FileName := ""
	err = db.QueryRow("SELECT GUID FROM Videos WHERE ID=?", VideoID).Scan(&FileName)
	if err != nil {
		log.Print("Database error: ", err)
	}
	return FileName
}

func getImageFileName(ImageID int) string {
	FileName := ""
	err = db.QueryRow("SELECT GUID FROM Images WHERE ID=?", ImageID).Scan(&FileName)
	if err != nil {
		log.Print("Database error: ", err)
	}
	return FileName
}

func uploadVideosLoop() {
	for {
		time.Sleep(5 * time.Second)
		VideoID := getVideoToUpload()
		if VideoID == 0 {
			continue
		}

		FileName := getVideoFileName(VideoID)
		TwitterMediaID := twitter.UploadFile(twitterClient, config.UploadPath+"/"+FileName)
		TimeNow := time.Now().Unix()
		_, err = db.Exec("UPDATE Videos SET MediaID=?,MediaTime=? WHERE ID=?", TwitterMediaID, TimeNow, VideoID)
		if err != nil {
			log.Print("Database error: ", err)
		}
	}
}

func uploadImagesLoop() {
	for {
		time.Sleep(5 * time.Second)
		ImageID := getImageToUpload()
		if ImageID == 0 {
			continue
		}

		FileName := getImageFileName(ImageID)
		TwitterMediaID := twitter.UploadFile(twitterClient, config.UploadPath+"/"+FileName)
		TimeNow := time.Now().Unix()
		_, err = db.Exec("UPDATE Images SET MediaID=?,MediaTime=? WHERE ID=?", TwitterMediaID, TimeNow, ImageID)
		if err != nil {
			log.Print("Database error: ", err)
		}
	}
}

func sendTweetLoop() {
	for {
		var VideoID int
		var ImageID int
		var TwitterMediaID string
		var Message string
		time.Sleep(5 * time.Second)
		TweetID := getTweetToSend()
		if TweetID == "0" {
			continue
		}

		err = db.QueryRow("SELECT Message, ImageA, VideoA FROM Tweets WHERE ID=?", TweetID).Scan(&Message, &ImageID, &VideoID)
		// tweet message with no image
		if (ImageID == 0) && (VideoID == 0) {
			twitterClient.StatusesUpdate(Message)
			markTweetAsSent(TweetID)
			log.Print("Tweet Sent")
			continue
		}

		if ImageID != 0 {
			err = db.QueryRow("SELECT MediaID FROM Images WHERE ID=?", ImageID).Scan(&TwitterMediaID)
		}

		if VideoID != 0 {
			err = db.QueryRow("SELECT MediaID FROM Videos WHERE ID=?", VideoID).Scan(&TwitterMediaID)
		}
		// tweet message with media
		if (VideoID != 0) && (TwitterMediaID != "") {
			twitterClient.StatusesUpdateWithMedia(Message, TwitterMediaID)
			markTweetAsSent(TweetID)
			log.Print("Tweet Sent with Video")
			continue
		}
		if (ImageID != 0) && (TwitterMediaID != "") {
			twitterClient.StatusesUpdateWithMedia(Message, TwitterMediaID)
			markTweetAsSent(TweetID)
			log.Print("Tweet Sent with Image")
			continue
		}
		// Something went wrong
		markTweetAsError(TweetID)
		log.Print("Tweet error")
	}
}

func main() {
	readConfig()
	databaseSetup()
	twitterSetup()

	var wg sync.WaitGroup
	wg.Add(1)

	go sendTweetLoop()
	go uploadImagesLoop()
	go uploadVideosLoop()

	wg.Wait()
}
