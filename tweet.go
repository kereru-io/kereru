package main

import _ "github.com/go-sql-driver/mysql"
import "github.com/gorilla/csrf"
import "strconv"
import "log"
import "net/http"
import "time"

func showNewTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	if (Access & NEWTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/tweet/new.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func submitNewTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	ImageID := "0"
	VideoID := "0"

	tweet := req.FormValue("tweet")
	Notes := req.FormValue("notes")
	TweetDate := req.FormValue("date")
	TweetTime := req.FormValue("time")

	MediaID := req.FormValue("MediaID")
	MediaType := req.FormValue("MediaType")

	if MediaID == "" {
		MediaID = "0"
	}

	if MediaType == "Image" {
		ImageID = MediaID
	}

	if MediaType == "Video" {
		VideoID = MediaID
	}

	TweetTimeObj := parseTime(TweetDate + " " + TweetTime)
	if TweetTimeObj.IsZero() {
		logicError(res, req, "Time value is not valid!")
		return
	}
	TweetUnixTime := TweetTimeObj.Unix()

	_, err = db.Exec("INSERT INTO Tweets(SendTime, Message, ImageA, VideoA, Notes, Status) VALUES(?, ?, ?, ?, ?, ?);", TweetUnixTime, tweet, ImageID, VideoID, Notes, "1")
	if err != nil {
		databaseError(res, req, err)
		return
	}

	var TweetID string = "0"
	err = db.QueryRow("SELECT ID from Tweets where SendTime=? and Message=?", TweetUnixTime, tweet).Scan(&TweetID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETDRAFT)

	http.Redirect(res, req, "/dashboard/tweets?Tweet="+TweetID+"#Tweet"+TweetID, http.StatusFound)
}

func showListTweets(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	TweetIndex, err := strconv.Atoi(req.FormValue("Tweet"))

	DisplayPagination := getUserDisplayPagination(req)
	DisplayDraft := getUserDisplayDraft(req)
	DisplayReviewed := getUserDisplayReviewed(req)
	DisplayReady := getUserDisplayReady(req)
	DisplaySent := getUserDisplaySent(req)
	DisplayDeleted := getUserDisplayDeleted(req)
	DisplayError := getUserDisplayError(req)
	DisplayFlagged := getUserDisplayFlagged(req)

	Select := "SELECT ID, SendTime, Message, ImageA, VideoA, Status FROM Tweets "
	Where := ""

	if DisplayDraft == 1 {
		Where = Where + "WHERE (Status & 1 = 1) "
	}

	if DisplayReviewed == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 2 = 2) "
		} else {
			Where = Where + "OR (Status & 2 = 2) "
		}
	}

	if DisplayReady == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 4 = 4) "
		} else {
			Where = Where + "OR (Status & 4 = 4) "
		}
	}

	if DisplaySent == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 8 = 8) "
		} else {
			Where = Where + "OR (Status & 8 = 8) "
		}
	}

	if DisplayDeleted == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 16 = 16) "
		} else {
			Where = Where + "OR (Status & 16 = 16) "
		}
	}

	if DisplayError == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 32 = 32) "
		} else {
			Where = Where + "OR (Status & 32 = 32) "
		}
	}

	if DisplayFlagged == 1 {
		if len(Where) == 0 {
			Where = Where + "WHERE (Status & 64 = 64) "
		} else {
			Where = Where + "OR (Status & 64 = 64) "
		}
	}

	if len(Where) == 0 {
		Where = "WHERE (ID=0) "
	}

	Search := Select + Where + "ORDER BY SendTime,ID "

	Page, err := strconv.Atoi(req.FormValue("page"))

	TotalTweets := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Tweets " + Where).Scan(&TotalTweets)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	if Page > (TotalTweets/100)+1 {
		Page = 1
	}

	if (Page == 0) && (TweetIndex > 0) {
		Page = getTweetPage(TweetIndex, Where)
	}

	if Page == 0 {
		Page = 1
	}

	if DisplayPagination == 1 {
		Search = Search + "LIMIT " + strconv.Itoa((Page-1)*100) + ", 100"
	}

	type Entry struct {
		TweetID   int
		Time      string
		Message   string
		MediaGUID string
		ImageID   string
		VideoID   string
		Status    string
	}

	var Tweets []Entry
	var Tweet Entry

	var TweetTime int64
	var Status int

	rows, err := db.Query(Search)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Tweet.TweetID, &TweetTime, &Tweet.Message, &Tweet.ImageID, &Tweet.VideoID, &Status)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		const DateFormat = "2006-01-02 at 15:04:05"
		TimeObj := time.Unix(TweetTime, 0)
		Tweet.Time = TimeObj.Format(DateFormat)
		if Tweet.ImageID != "0" {
			Tweet.MediaGUID = getGUIDForImage(Tweet.ImageID)
		}
		if Tweet.VideoID != "0" {
			Tweet.MediaGUID = getGUIDForVideo(Tweet.VideoID)
		}
		Tweet.Status = getStatusAsString(Status)
		Tweets = append(Tweets, Tweet)
	}

	PageLast := int(TotalTweets/100) + 1
	PagePre := Page - 1
	PageNext := Page + 1

	if PageNext > PageLast {
		PageNext = PageLast
	}
	if PagePre < 1 {
		PagePre = 1
	}

	vars := map[string]interface{}{
		"UserID":             getUsername(req),
		"AccessNewTweet":     (Access & NEWTWEET),
		"AccessEditTweet":    (Access & EDITTWEET),
		"AccessReviewTweet":  (Access & REVIEWTWEET),
		"AccessPublishTweet": (Access & PUBLISHTWEET),
		"AccessFlagTweet":    (Access & FLAGTWEET),
		"AccessDeleteTweet":  (Access & DELETETWEET),
		"AccessUploadImage":  (Access & NEWIMAGE),
		"AccessUploadVideo":  (Access & NEWVIDEO),
		"DisplayPagination":  DisplayPagination,
		"PagePre":            PagePre,
		"Page":               Page,
		"PageNext":           PageNext,
		"PageLast":           PageLast,
		csrf.TemplateTag:     csrf.TemplateField(req),
		"Tweets":             Tweets,
	}

	t := createHTML("/tweet/list.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showEditTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	tweetID := req.FormValue("ID")
	var ID int
	var Message string
	var ImageA string
	var VideoA string
	var TweetTime int64
	var Notes string
	var Status int

	row := db.QueryRow("SELECT ID, SendTime, Message, ImageA, VideoA, Notes, Status FROM Tweets WHERE ID=?", tweetID)
	row.Scan(&ID, &TweetTime, &Message, &ImageA, &VideoA, &Notes, &Status)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	TweetTimeObj := time.Unix(TweetTime, 0)
	const DateFormat = "2006-01-02"
	const TimeFormat = "15:04:05"

	MediaID := ""
	MediaType := "Image"
	MediaGUID := ""

	if ImageA != "0" {
		MediaID = ImageA
		MediaType = "Image"
		MediaGUID = getGUIDForImage(ImageA)
	}
	if VideoA != "0" {
		MediaID = VideoA
		MediaType = "Video"
		MediaGUID = getGUIDForVideo(VideoA)
	}

	vars := map[string]interface{}{
		"UserID":             getUsername(req),
		"AccessNewTweet":     (Access & NEWTWEET),
		"AccessEditTweet":    (Access & EDITTWEET),
		"AccessReviewTweet":  (Access & REVIEWTWEET),
		"AccessPublishTweet": (Access & PUBLISHTWEET),
		"AccessFlagTweet":    (Access & FLAGTWEET),
		"AccessUploadImage":  (Access & NEWIMAGE),
		"AccessUploadVideo":  (Access & NEWVIDEO),
		"TweetID":            ID,
		"Date":               TweetTimeObj.Format(DateFormat),
		"Time":               TweetTimeObj.Format(TimeFormat),
		"Message":            Message,
		"MediaID":            MediaID,
		"MediaGUID":          MediaGUID,
		"MediaType":          MediaType,
		"Notes":              Notes,
		csrf.TemplateTag:     csrf.TemplateField(req),
	}

	t := createHTML("/tweet/edit.tmpl")
	t.Execute(res, vars)
}

func submitEditTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	ImageID := "0"
	VideoID := "0"

	TweetID := req.FormValue("TweetID")
	tweet := req.FormValue("tweet")
	Notes := req.FormValue("notes")
	TweetDate := req.FormValue("date")
	TweetTime := req.FormValue("time")

	MediaID := req.FormValue("MediaID")
	MediaType := req.FormValue("MediaType")

	if MediaID == "" {
		MediaID = "0"
	}

	if MediaType == "Image" {
		ImageID = MediaID
	}

	if MediaType == "Video" {
		VideoID = MediaID
	}

	TweetTimeObj := parseTime(TweetDate + " " + TweetTime)
	if TweetTimeObj.IsZero() {
		logicError(res, req, "Time value is not valid!")
		return
	}
	TweetUnixTime := TweetTimeObj.Unix()

	_, err = db.Exec("UPDATE Tweets SET SendTime=?, Message=?, ImageA=?, VideoA=?, Notes=?, Status=1 WHERE ID=?", TweetUnixTime, tweet, ImageID, VideoID, Notes, TweetID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETDRAFT)

	http.Redirect(res, req, "/dashboard/tweets?Tweet="+TweetID+"#Tweet"+TweetID, http.StatusFound)
}

func submitDeleteTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & DELETETWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	TweetID := req.FormValue("ID")

	_, err = db.Exec("UPDATE Tweets SET Status=16 WHERE ID=?", TweetID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETDELETED)

	http.Redirect(res, req, "/dashboard/home", http.StatusFound)
}

func showStatusTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if ((Access & REVIEWTWEET) == 0) && ((Access & PUBLISHTWEET) == 0) && ((Access & FLAGTWEET) == 0) && ((Access & DELETETWEET) == 0) {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	tweetID := req.FormValue("ID")
	var ID int
	var Message string
	var ImageA string
	var VideoA string
	var TweetTime int64
	var Notes string
	var Status int
	var Action string

	row := db.QueryRow("SELECT ID, SendTime, Message, ImageA, VideoA, Notes, Status FROM Tweets WHERE ID=?", tweetID)
	row.Scan(&ID, &TweetTime, &Message, &ImageA, &VideoA, &Notes, &Status)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	TweetTimeObj := time.Unix(TweetTime, 0)
	const DateFormat = "2006-01-02"
	const TimeFormat = "15:04:05"

	MediaID := ""
	MediaType := "Image"
	MediaGUID := "0"

	if ImageA != "0" {
		MediaID = ImageA
		MediaType = "Image"
		MediaGUID = getGUIDForImage(ImageA)
	}
	if VideoA != "0" {
		MediaID = VideoA
		MediaType = "Video"
		MediaGUID = getGUIDForVideo(VideoA)
	}

	if (Status & 1) == 1 {
		Action = "reviewed"
	}

	if (Status & 2) == 2 {
		Action = "published"
	}

	vars := map[string]interface{}{
		"UserID":             getUsername(req),
		"AccessNewTweet":     (Access & NEWTWEET),
		"AccessEditTweet":    (Access & EDITTWEET),
		"AccessReviewTweet":  (Access & REVIEWTWEET),
		"AccessPublishTweet": (Access & PUBLISHTWEET),
		"AccessFlagTweet":    (Access & FLAGTWEET),
		"AccessDeleteTweet":  (Access & DELETETWEET),
		"AccessUploadImage":  (Access & NEWIMAGE),
		"AccessUploadVideo":  (Access & NEWVIDEO),
		"AccessAuditTweet":   (Access & AUDITTWEET),
		"ID":                 ID,
		"Date":               TweetTimeObj.Format(DateFormat),
		"Time":               TweetTimeObj.Format(TimeFormat),
		"Message":            Message,
		"MediaID":            MediaID,
		"MediaGUID":          MediaGUID,
		"MediaType":          MediaType,
		"Notes":              Notes,
		"Action":             Action,
		csrf.TemplateTag:     csrf.TemplateField(req),
	}

	t := createHTML("/tweet/status.tmpl")
	t.Execute(res, vars)
}

func submitPublishTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & PUBLISHTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	TweetID := req.FormValue("ID")
	Action := req.FormValue("ACTION")

	if Action == "publish" {
		_, err = db.Exec("UPDATE Tweets SET Status=4 WHERE ID=?", TweetID)
		if err != nil {
			databaseError(res, req, err)
			return
		}
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETREADY)

	http.Redirect(res, req, "/dashboard/tweets?Tweet="+TweetID+"#Tweet"+TweetID, http.StatusFound)
}

func submitReviewTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & REVIEWTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	TweetID := req.FormValue("ID")
	Action := req.FormValue("ACTION")

	if Action == "reviewed" {
		_, err = db.Exec("UPDATE Tweets SET Status=2 WHERE ID=?", TweetID)
		if err != nil {
			databaseError(res, req, err)
			return
		}
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETREVIEWED)

	http.Redirect(res, req, "/dashboard/tweets?Tweet="+TweetID+"#Tweet"+TweetID, http.StatusFound)
}

func submitFlaggedTweet(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & FLAGTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	TweetID := req.FormValue("ID")

	_, err = db.Exec("UPDATE Tweets SET Status=64 WHERE ID=?", TweetID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	UserID := getUserID(req)
	tweetAuditEvent(UserID, TweetID, TWEETFLAGGED)

	http.Redirect(res, req, "/dashboard/tweets?Tweet="+TweetID+"#Tweet"+TweetID, http.StatusFound)
}

func showTweetAudit(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & AUDITTWEET) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	TweetID := req.FormValue("ID")

	type Entry struct {
		Time   string
		User   string
		Status string
	}

	var Time int64
	var Status int
	var UserID int
	var Event Entry
	var Events []Entry

	// TweetAudit(UserID, Time, TweetID, Status)
	rows, err := db.Query("SELECT Time, UserID, Status FROM TweetAudit WHERE TweetID=? ORDER BY Time DESC", TweetID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Time, &UserID, &Status)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		const DateFormat = "2006-01-02 at 15:04:05"
		TimeObj := time.Unix(Time, 0)
		Event.Time = TimeObj.Format(DateFormat)
		Event.Status = getStatusAsString(Status)
		Event.User = getUserNameFromUserID(UserID)

		Events = append(Events, Event)
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"Events":            Events,
		csrf.TemplateTag:    csrf.TemplateField(req),
	}

	t := createHTML("/tweet/audit.tmpl")
	t.Execute(res, vars)
}
