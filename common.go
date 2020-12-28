package main

import _ "github.com/go-sql-driver/mysql"
import "time"
import "log"

const requiredDBVersion = 1

//Tweet Status magic numbers
const (
	TWEETDRAFT    int = 1
	TWEETREVIEWED int = 2
	TWEETREADY    int = 4
	TWEETSENT     int = 8
	TWEETDELETED  int = 16
	TWEETERRORED  int = 32
	TWEETFLAGGED  int = 64
)

func tweetAuditEvent(UserID int, TweetID string, Status int) {
	EventTime := time.Now().Unix()
	_, err = db.Exec("INSERT INTO TweetAudit(UserID, Time, TweetID, Status) VALUES(?, ?, ?, ?);", UserID, EventTime, TweetID, Status)
	if err != nil {
		log.Print("Can't write to data audit log!")
	}
}
