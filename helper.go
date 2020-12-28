package main

import "github.com/go-sql-driver/mysql"
import "log"
import "fmt"
import "runtime"
import "net/http"
import "time"
import "strconv"
import "html/template"
import "strings"
import "io/ioutil"

func migrateDatabase(DatabaseVersion int, DatabaseRequired int) {
	log.Print("Upgrading database from version ", DatabaseVersion, " to version ", DatabaseRequired)
	for DatabaseVersion < DatabaseRequired {
		DatabaseVersion = DatabaseVersion + 1
		log.Print("Conversion step: ", DatabaseVersion)
		file, err := ioutil.ReadFile(config.WebRoot + "/schema/mysql/" + strconv.Itoa(DatabaseVersion) + ".sql")
		if err != nil {
			log.Print("Can't load default sql schema!")
			log.Fatal(err)
		}
		requests := strings.Split(string(file), ";")
		for _, request := range requests {
			_, err = db.Exec(request)
			if err != nil {
				SQL_error := err.(*mysql.MySQLError).Number
				if SQL_error != 1065 {
					log.Print("Can't process default sql schema!")
					log.Print(request)
					log.Fatal(err)
				}
			}
		}
		_, err = db.Exec("INSERT INTO Migration (Version) values (?)", DatabaseVersion)
		if err != nil {
			log.Print("Migration Error!")
			log.Fatal(err)
		}
	}
}

func getStatusAsString(StatusValue int) string {
	StatusString := ""
	if (StatusValue & TWEETDRAFT) != 0 {
		StatusString = "Draft"
	}

	if (StatusValue & TWEETREVIEWED) != 0 {
		StatusString = "Reviewed"
	}

	if (StatusValue & TWEETREADY) != 0 {
		StatusString = "Ready"
	}

	if (StatusValue & TWEETSENT) != 0 {
		StatusString = "Sent"
	}

	if (StatusValue & TWEETDELETED) != 0 {
		StatusString = "Deleted"
	}

	if (StatusValue & TWEETERRORED) != 0 {
		StatusString = "Errored"
	}

	if (StatusValue & TWEETFLAGGED) != 0 {
		StatusString = "Flagged"
	}
	return StatusString
}

func getRoleAsString(Access int) string {
	AccessList := ""
	if (Access & NEWUSER) != 0 {
		AccessList = AccessList + "Add New Users, "
	}
	if (Access & EDITUSER) != 0 {
		AccessList = AccessList + "Edit Users, "
	}
	if (Access & RBAC) != 0 {
		AccessList = AccessList + "Role Editor, "
	}
	if (Access & NEWTWEET) != 0 {
		AccessList = AccessList + "New Tweets, "
	}
	if (Access & EDITTWEET) != 0 {
		AccessList = AccessList + "Edit Tweets, "
	}
	if (Access & REVIEWTWEET) != 0 {
		AccessList = AccessList + "Review Tweets, "
	}
	if (Access & PUBLISHTWEET) != 0 {
		AccessList = AccessList + "Publish Tweets, "
	}
	if (Access & FLAGTWEET) != 0 {
		AccessList = AccessList + "Flag Tweets, "
	}
	if (Access & DELETETWEET) != 0 {
		AccessList = AccessList + "Delete Tweets, "
	}
	if (Access & NEWIMAGE) != 0 {
		AccessList = AccessList + "New Images, "
	}
	if (Access & EDITIMAGE) != 0 {
		AccessList = AccessList + "Edit Images, "
	}
	if (Access & DELETEIMAGE) != 0 {
		AccessList = AccessList + "Delete Images, "
	}
	if (Access & NEWVIDEO) != 0 {
		AccessList = AccessList + "New Video, "
	}
	if (Access & EDITVIDEO) != 0 {
		AccessList = AccessList + "Edit Video, "
	}
	if (Access & DELETEVIDEO) != 0 {
		AccessList = AccessList + "Delete Video, "
	}
	if (Access & AUDITTWEET) != 0 {
		AccessList = AccessList + "View Tweet Audit, "
	}
	return AccessList
}

func getTweetPage(TweetIndex int, Where string) int {
	I := 0
	var TweetID int = 0

	rows, err := db.Query("SELECT ID FROM Tweets " + Where + " ORDER BY SendTime,ID ")
	if err != nil {
		return 1
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&TweetID)
		if err != nil {
			return 1
		}
		if TweetID == TweetIndex {
			return (I / 100) + 1
		}
		I = I + 1
	}
	return 1
}

func getImagePage(ImageIndex int) int {
	I := 0
	var ImageID int = 0

	rows, err := db.Query("SELECT ID FROM images ORDER BY time DESC")
	if err != nil {
		return 1
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&ImageID)
		if err != nil {
			return 1
		}
		if ImageID == ImageIndex {
			return (I / 100) + 1
		}
		I = I + 1
	}
	return 1
}

func getVideoPage(VideoIndex int) int {
	I := 0
	var VideoID int = 0

	rows, err := db.Query("SELECT ID FROM Videos ORDER BY Time DESC")
	if err != nil {
		return 1
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&VideoID)
		if err != nil {
			return 1
		}
		if VideoID == VideoIndex {
			return (I / 100) + 1
		}
		I = I + 1
	}
	return 1
}

func getGUIDForImage(ImageID string) string {
	var GUID string = "0"
	_ = db.QueryRow("SELECT GUID FROM Images WHERE ID=?", ImageID).Scan(&GUID)
	return GUID
}

func getGUIDForVideo(VideoID string) string {
	var GUID string = "0"
	_ = db.QueryRow("SELECT GUID FROM Videos WHERE ID=?", VideoID).Scan(&GUID)
	return GUID
}

func getUserNameFromUserID(UserID int) string {
	var UserName = ""
	_ = db.QueryRow("SELECT Username FROM Users WHERE ID=?", UserID).Scan(&UserName)
	return UserName
}

func loggingMw(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request: %v %v %v %v %v %v", r.RemoteAddr, r.Method, r.RequestURI, r.Proto, r.ContentLength, r.Host)
			next.ServeHTTP(w, r)
		})
}

func getUsername(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	username := sessionStore[cookie.Value].Username
	return username
}

func getUserID(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	UserID := sessionStore[cookie.Value].UserID
	return UserID
}

func getUserAccess(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	AccessLevel := sessionStore[cookie.Value].AccessLevel
	return AccessLevel
}

func logicError(res http.ResponseWriter, req *http.Request, UserError string) {
	cookie, err := req.Cookie("session")
	if err != nil {
		return
	}
	Error := ""

	_, File, LineNum, _ := runtime.Caller(1)

	if config.DebugLevel > 1 {
		log.Printf("Logic error: %v\nIn file: %v\nNear line %v", UserError, File, LineNum)
	} else {
		log.Printf("Logic error!")
	}

	if config.DebugLevel > 3 {
		Error = fmt.Sprintf("Logic error: %v\nIn file: %v\nNear line %v", UserError, File, LineNum)
	} else {
		Error = fmt.Sprintf("A logic error has happened - See system logs for more infomation.")
	}

	var UserState = sessionStore[cookie.Value]
	UserState.LastError = Error
	sessionStore[cookie.Value] = UserState
	http.Redirect(res, req, "/dashboard/error", http.StatusFound)
	return
}

func databaseError(res http.ResponseWriter, req *http.Request, UserError error) {
	cookie, err := req.Cookie("session")
	if err != nil {
		return
	}
	Error := ""

	_, File, LineNum, _ := runtime.Caller(1)

	if config.DebugLevel > 1 {
		log.Printf("Database error: %v\nIn file: %v\nNear line %v", UserError.Error(), File, LineNum)
	} else {
		log.Printf("Database error")
	}

	if config.DebugLevel > 3 {
		Error = fmt.Sprintf("Database error: %v\nIn file: %v\nNear line %v", UserError.Error(), File, LineNum)
	} else {
		Error = fmt.Sprintf("A Database error has happened - See system logs for more infomation.")
	}

	var UserState = sessionStore[cookie.Value]
	UserState.LastError = Error
	sessionStore[cookie.Value] = UserState
	http.Redirect(res, req, "/dashboard/error", http.StatusFound)
	return
}

func getUserLastError(r *http.Request) string {
	cookie, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return sessionStore[cookie.Value].LastError
}

func getUserDisplayPagination(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Pagination := sessionStore[cookie.Value].DisplayPagination
	return Pagination
}

func getUserDisplaySent(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Sent := sessionStore[cookie.Value].DisplaySent
	return Sent
}

func getUserDisplayDraft(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Draft := sessionStore[cookie.Value].DisplayDraft
	return Draft
}

func getUserDisplayReviewed(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Reviewed := sessionStore[cookie.Value].DisplayReviewed
	return Reviewed
}

func getUserDisplayReady(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Ready := sessionStore[cookie.Value].DisplayReady
	return Ready
}

func getUserDisplayFlagged(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Flagged := sessionStore[cookie.Value].DisplayFlagged
	return Flagged
}

func getUserDisplayDeleted(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Deleted := sessionStore[cookie.Value].DisplayDeleted
	return Deleted
}

func getUserDisplayError(r *http.Request) int {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0
	}
	Error := sessionStore[cookie.Value].DisplayError
	return Error
}

func parseTime(input string) time.Time {
	var TimeObj time.Time
	TimeObj, _ = time.Parse("2006-01-02 15:04:05", input)
	if !TimeObj.IsZero() {
		return TimeObj
	}
	TimeObj, _ = time.Parse("2006-01-02 15:04", input)
	if !TimeObj.IsZero() {
		return TimeObj
	}
	return TimeObj
}

func authMw(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			if totalUsers == 0 {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if cookie == nil {
			if totalUsers == 0 {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if sessionStore[cookie.Value].Username == "" {
			if totalUsers == 0 {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func createHTML(Page string) *template.Template {
	html := template.Must(template.ParseFiles(config.WebRoot+"/templates/base.tmpl", config.WebRoot+"/templates/nav.tmpl", config.WebRoot+"/templates/"+Page))
	return html
}
