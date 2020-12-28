package main

import "fmt"
import "os"
import "database/sql"
import _ "github.com/go-sql-driver/mysql"
import "github.com/gorilla/csrf"
import "github.com/gorilla/mux"
import _ "crypto/tls"
import "log"
import "net/http"
import "time"

var db *sql.DB
var err error // internal error managment
var sessionStore map[string]client
var totalUsers int

type client struct {
	loggedIn          bool
	Username          string
	UserID            int
	LastError         string
	AccessLevel       int
	DisplayPagination int
	DisplayDraft      int
	DisplayReviewed   int
	DisplayReady      int
	DisplaySent       int
	DisplayDeleted    int
	DisplayError      int
	DisplayFlagged    int
}

func main() {
	readConfig()

	if _, err = os.Stat(config.UploadPath); os.IsNotExist(err) {
		err = os.MkdirAll(config.UploadPath, 0644)
		if err != nil {
			log.Fatal("Cant create uploads: ", err)
		}
	}

	// setup the map of clent data / cookie
	sessionStore = make(map[string]client)

	// Create an sql.DB and check for errors
	db, err = sql.Open("mysql", fmt.Sprintf("%v:%v@/%v?charset=utf8mb4", config.DatabaseUser, config.DatabasePassword, config.DatabaseName))
	if err != nil {
		panic(err.Error())
	}

	// Test the connection to the database
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// sql.DB should be long lived "defer" closes it at function end
	defer db.Close()

	currentDBVersion := 0
	err = db.QueryRow("SELECT Version from Migration ORDER BY Version DESC limit 1").Scan(&currentDBVersion)

	if currentDBVersion < requiredDBVersion {
		migrateDatabase(currentDBVersion, requiredDBVersion)
	}

	if currentDBVersion > requiredDBVersion {
		log.Fatal("Wrong Application version!")
		return
	}

	err = db.QueryRow("SELECT COUNT(*) FROM Users").Scan(&totalUsers)

	CSRF := csrf.Protect(
		[]byte(config.CsrfToken),
		csrf.RequestHeader("Authenticity-Token"),
		csrf.FieldName("authenticity_token"),
		csrf.Secure(config.SecureCookie),
		//  csrf.ErrorHandler(http.HandlerFunc(serverError(403))),
	)

	//setup page handlers
	web := mux.NewRouter()
	web.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/home", http.StatusFound)
	})

	web.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir(config.WebRoot+"/css"))))
	web.PathPrefix("/js/").Handler(http.StripPrefix("/js/", http.FileServer(http.Dir(config.WebRoot+"/js"))))

	web.HandleFunc("/login", showLogin)
	web.HandleFunc("/login/post", submitLogin)

	web.HandleFunc("/setup", showSetup)
	web.HandleFunc("/setup/post", submitSetup)

	web.HandleFunc("/forgot", showForgotPwd)
	web.HandleFunc("/forgot/post", submitForgotPwd)
	web.HandleFunc("/pwreset", showResetPwd)
	web.HandleFunc("/pwreset/post", submitResetPwd)

	dashboard := web.PathPrefix("/dashboard").Subrouter()
	dashboard.HandleFunc("/error", showErrorPage)

	dashboard.HandleFunc("/home", showListTweets)

	dashboard.HandleFunc("/rbac", showListRBAC)
	dashboard.HandleFunc("/rbac/new", showNewRBAC)
	dashboard.HandleFunc("/rbac/new/post", submitNewRBAC)
	dashboard.HandleFunc("/rbac/edit", showEditRBAC)
	dashboard.HandleFunc("/rbac/edit/post", submitEditRBAC)
	dashboard.HandleFunc("/rbac/delete/post", submitDeleteRBAC)

	dashboard.HandleFunc("/users", showListUsers)
	dashboard.HandleFunc("/users/edit", showEditUser)
	dashboard.HandleFunc("/users/edit/post", submitEditUser)
	dashboard.HandleFunc("/users/delete/post", submitDeleteUser)
	dashboard.HandleFunc("/users/new", showNewUser)
	dashboard.HandleFunc("/users/new/post", submitNewUser)

	dashboard.HandleFunc("/change/display", showDisplay)
	dashboard.HandleFunc("/change/display/post", submitDisplay)

	dashboard.HandleFunc("/user/settings", showSettingsUser)
	dashboard.HandleFunc("/user/pwd", showChangePwd)
	dashboard.HandleFunc("/user/pwd/post", submitChangePwd)
	dashboard.HandleFunc("/user/email", showChangeEmail)
	dashboard.HandleFunc("/user/email/post", submitChangeEmail)

	dashboard.HandleFunc("/tweets", showListTweets)
	dashboard.HandleFunc("/tweets/new", showNewTweet)
	dashboard.HandleFunc("/tweets/new/post", submitNewTweet)
	dashboard.HandleFunc("/tweets/edit", showEditTweet)
	dashboard.HandleFunc("/tweets/edit/post", submitEditTweet)
	dashboard.HandleFunc("/tweets/delete/post", submitDeleteTweet)
	dashboard.HandleFunc("/tweets/status", showStatusTweet)
	dashboard.HandleFunc("/tweets/review/post", submitReviewTweet)
	dashboard.HandleFunc("/tweets/publish/post", submitPublishTweet)
	dashboard.HandleFunc("/tweets/flag/post", submitFlaggedTweet)
	dashboard.HandleFunc("/tweets/audit", showTweetAudit)

	dashboard.PathPrefix("/media/view").Handler(http.StripPrefix("/dashboard/media/view/", http.FileServer(http.Dir(config.UploadPath))))

	dashboard.HandleFunc("/images", showListImages)
	dashboard.HandleFunc("/images/new", showNewImage)
	dashboard.HandleFunc("/images/new/post", submitNewImage)
	dashboard.HandleFunc("/images/edit", showEditImage)
	dashboard.HandleFunc("/images/edit/post", submitEditImage)
	dashboard.HandleFunc("/images/delete/post", submitDeleteImage)
	dashboard.HandleFunc("/images/resize", showImageResized)
	dashboard.HandleFunc("/images/resize/post", submitImageResized)
	dashboard.HandleFunc("/images/error", showImageError)

	dashboard.HandleFunc("/images/one.json", getOneImg)
	dashboard.HandleFunc("/images/list.json", getListImg)
	dashboard.HandleFunc("/images/pagecount.json", getImgPageCount)

	dashboard.HandleFunc("/videos", showListVideos)
	dashboard.HandleFunc("/videos/new", showNewVideo)
	dashboard.HandleFunc("/videos/new/post", submitNewVideo)
	dashboard.HandleFunc("/videos/edit", showEditVideo)
	dashboard.HandleFunc("/videos/edit/post", submitEditVideo)
	dashboard.HandleFunc("/videos/delete/post", submitDeleteVideo)
	dashboard.HandleFunc("/videos/error", showVideoError)

	dashboard.HandleFunc("/videos/one.json", getOneVid)
	dashboard.HandleFunc("/videos/list.json", getListVid)
	dashboard.HandleFunc("/videos/pagecount.json", getVidPageCount)

	dashboard.HandleFunc("/logout", logoutPage)

	web.Use(loggingMw)
	dashboard.Use(authMw)

	server := &http.Server{
		Addr:           ":" + config.WebPort,
		Handler:        CSRF(web),
		ReadTimeout:    20 * time.Second,
		WriteTimeout:   20 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
		//	TLSNextProto:   map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	if config.TLS == true {
		err = server.ListenAndServeTLS(config.Cert, config.Key)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func showErrorPage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"error":             getUserLastError(req),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/error.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Fatal(err)
	}
}
