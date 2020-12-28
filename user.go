package main

import "bytes"
import "database/sql"
import _ "github.com/go-sql-driver/mysql"
import "golang.org/x/crypto/bcrypt"
import "github.com/satori/go.uuid"
import "github.com/gorilla/csrf"
import "log"
import "strconv"
import "net/http"
import "net/smtp"
import "time"

func showSetup(res http.ResponseWriter, req *http.Request) {
	if totalUsers != 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	vars := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
	}

	t := createHTML("/user/setup.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitSetup(res http.ResponseWriter, req *http.Request) {
	if totalUsers != 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	username := req.FormValue("username")
	firstname := req.FormValue("firstname")
	lastname := req.FormValue("lastname")
	emailAddress := req.FormValue("email")
	password := req.FormValue("passwordA")
	role := 2 //admin role

	var user string

	err := db.QueryRow("SELECT Username FROM Users WHERE Username=?", username).Scan(&user)
	switch {
	case err == sql.ErrNoRows: // Username is available
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}

		_, err = db.Exec("INSERT INTO Users(Username, Password, EmailAddress, Role, FirstName, LastName ) VALUES(?, ?, ?, ?, ?, ?)", username, hashedPassword, emailAddress, role, firstname, lastname)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		totalUsers = 1
		http.Redirect(res, req, "/dashboard/user/settings", http.StatusFound)
		return

	case err != nil:
		databaseError(res, req, err)
		return

	default:
		logicError(res, req, "Username in use")
		return
	}
}

func showDisplay(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	DisplayPagination := getUserDisplayPagination(req)
	DisplayDraft := getUserDisplayDraft(req)
	DisplayReviewed := getUserDisplayReviewed(req)
	DisplayReady := getUserDisplayReady(req)
	DisplaySent := getUserDisplaySent(req)
	DisplayDeleted := getUserDisplayDeleted(req)
	DisplayError := getUserDisplayError(req)
	DisplayFlagged := getUserDisplayFlagged(req)

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),

		"DisplayDraft":      DisplayDraft,
		"DisplayReviewed":   DisplayReviewed,
		"DisplayReady":      DisplayReady,
		"DisplaySent":       DisplaySent,
		"DisplayError":      DisplayError,
		"DisplayDeleted":    DisplayDeleted,
		"DisplayFlagged":    DisplayFlagged,
		"DisplayPagination": DisplayPagination,

		csrf.TemplateTag: csrf.TemplateField(req),
	}
	t := createHTML("/user/display.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitDisplay(res http.ResponseWriter, req *http.Request) {
	DatabaseUsername := getUsername(req)
	DatabaseUserID := getUserID(req)
	AccessLevel := getUserAccess(req)
	DisplayPagination, _ := strconv.Atoi(req.FormValue("pagination"))
	DisplayDraft, _ := strconv.Atoi(req.FormValue("showdraft"))
	DisplayReviewed, _ := strconv.Atoi(req.FormValue("showreviewed"))
	DisplayReady, _ := strconv.Atoi(req.FormValue("showready"))
	DisplaySent, _ := strconv.Atoi(req.FormValue("showsent"))
	DisplayDeleted, _ := strconv.Atoi(req.FormValue("showdeleted"))
	DisplayError, _ := strconv.Atoi(req.FormValue("showerror"))
	DisplayFlagged, _ := strconv.Atoi(req.FormValue("showflagged"))

	cookie, _ := req.Cookie("session")

	sessionStore[cookie.Value] = client{
		false,
		DatabaseUsername,
		DatabaseUserID,
		"What error?",
		AccessLevel,
		DisplayPagination,
		DisplayDraft,
		DisplayReviewed,
		DisplayReady,
		DisplaySent,
		DisplayDeleted,
		DisplayError,
		DisplayFlagged}
	http.Redirect(res, req, "/dashboard/tweets?page=Home", http.StatusFound)
}

func sendResetEmail(EmailAddress string, Token string) {

	c, err := smtp.Dial("127.0.0.1:25")
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
	defer c.Close()

	// Set the sender and recipient.
	c.Mail("no-reply@" + config.WebHost)
	c.Rcpt(EmailAddress)

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
	defer wc.Close()

	buf := bytes.NewBufferString(
		"To: " + EmailAddress + "\r\n" +
			"From: no-reply@" + config.WebHost + "\r\n" +
			"Subject: Password reset\r\n" +
			"\r\n" +
			"To reset your password use this link:\r\n" +
			"https://" + config.WebHost + "/pwreset?Token=" +
			Token + "&Email=" + EmailAddress + " \r\n" +
			"\r\n" +
			"\r\n")

	if _, err = buf.WriteTo(wc); err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showSettingsUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewUser":     (Access & NEWUSER),
		"AccessEditUser":    (Access & EDITUSER),
		"AccessRBAC":        (Access & RBAC),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/user/settings.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func logoutPage(res http.ResponseWriter, req *http.Request) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   config.SecureCookie,
	}

	http.SetCookie(res, cookie)
	http.Redirect(res, req, "/dashboard/home", http.StatusFound)
}

func showLogin(res http.ResponseWriter, req *http.Request) {
	vars := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
	}
	t := createHTML("/user/login.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showChangePwd(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/user/change_pwd.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showForgotPwd(res http.ResponseWriter, req *http.Request) {
	vars := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
	}
	t := createHTML("/user/forgot_pwd.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitForgotPwd(res http.ResponseWriter, req *http.Request) {
	var databaseUsername string
	var databaseEmail string
	var databaseID int
	var token string

	// Grab the email address from the submitted post form
	email := req.FormValue("email")

	err := db.QueryRow("SELECT ID, Username, EmailAddress FROM Users WHERE EmailAddress=?", email).Scan(&databaseID, &databaseUsername, &databaseEmail)
	if err != nil {
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	uuidtoken := uuid.NewV4()
	token = uuidtoken.String()

	TokenTime := time.Now().Unix()

	_, err = db.Exec("INSERT INTO PasswordResets(UID, Email, Token, ResetTime) VALUES(?, ?, ?, ?)", databaseID, databaseEmail, token, TokenTime)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	sendResetEmail(email, token)
	http.Redirect(res, req, "/login", http.StatusFound)
	return
}

func showResetPwd(res http.ResponseWriter, req *http.Request) {
	UserToken := req.FormValue("Token")
	UserEmail := req.FormValue("Email")
	var DBToken string
	var DBEmail string
	var DBTokenTime int64

	vars := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
		"Token":          UserToken,
		"Email":          UserEmail,
	}

	err := db.QueryRow("SELECT Email, Token, ResetTime FROM PasswordResets WHERE (Token=?) and (Email=?)", UserToken, UserEmail).Scan(&DBEmail, &DBToken, &DBTokenTime)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	TokenTime := time.Now().Unix() - 600

	if (DBToken == UserToken) && (DBEmail == UserEmail) && (DBTokenTime > TokenTime) {
		t := createHTML("/user/pwreset.tmpl")
		err := t.Execute(res, vars)
		if err != nil {
			log.Print("Something went wrong: %s", err)
			return
		}
		return
	}
	http.Redirect(res, req, "/login", http.StatusFound)
	return
}

func submitResetPwd(res http.ResponseWriter, req *http.Request) {
	UserToken := req.FormValue("token")
	UserEmail := req.FormValue("email")
	UserPasswordA := req.FormValue("passwordA")
	UserPasswordB := req.FormValue("passwordB")
	var DBToken string
	var DBEmail string
	var DBUID string
	var DBTokenTime int64

	err := db.QueryRow("SELECT UID, Email, Token, ResetTime FROM PasswordResets WHERE (Token=?) and (Email=?)", UserToken, UserEmail).Scan(&DBUID, &DBEmail, &DBToken, &DBTokenTime)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	TokenTime := time.Now().Unix() - 600

	if (DBToken == UserToken) && (DBEmail == UserEmail) && (DBTokenTime > TokenTime) {
		if UserPasswordA == UserPasswordB {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(UserPasswordA), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Something went wrong: %s", err)
				return
			}
			_, err = db.Exec("UPDATE Users SET Password=? WHERE ID=?", hashedPassword, DBUID)
			if err != nil {
				databaseError(res, req, err)
				return
			}
		}
	}
	http.Redirect(res, req, "/login", http.StatusFound)
	return
}

func submitLogin(res http.ResponseWriter, req *http.Request) {
	Username := req.FormValue("username")
	Password := req.FormValue("password")

	var DatabaseUsername string
	var DatabasePassword string
	var DatabaseUID int
	var Role int
	var AccessLevel int

	// Search the database for the username provided
	err := db.QueryRow("SELECT ID, Username, Password, Role FROM Users WHERE Username=?", Username).Scan(&DatabaseUID, &DatabaseUsername, &DatabasePassword, &Role)

	// If not then redirect to the login page
	if err != nil {
		log.Printf("Bad Username: %s", Username)
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	// Validate the password
	err = bcrypt.CompareHashAndPassword([]byte(DatabasePassword), []byte(Password))

	// If wrong password redirect to the login
	if err != nil {
		log.Printf("Bad password for: %s", Username)
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	// If the login succeeded
	uuidtoken := uuid.NewV4()
	token := uuidtoken.String()

	cookie := &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   config.SecureCookie,
	}

	err = db.QueryRow("SELECT Access FROM Roles WHERE RoleID=?", Role).Scan(&AccessLevel)
	if err != nil {
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	//load display defaults from database.
	DisplayPagination := 1
	DisplayDraft := 1
	DisplayReviewed := 1
	DisplayReady := 1
	DisplaySent := 0
	DisplayDeleted := 0
	DisplayError := 1
	DisplayFlagged := 1

	sessionStore[cookie.Value] = client{
		false,
		DatabaseUsername,
		DatabaseUID,
		"What error?",
		AccessLevel,
		DisplayPagination,
		DisplayDraft,
		DisplayReviewed,
		DisplayReady,
		DisplaySent,
		DisplayDeleted,
		DisplayError,
		DisplayFlagged,
	}

	http.SetCookie(res, cookie)
	http.Redirect(res, req, "/dashboard/tweets?page=Home", http.StatusFound)
}

func submitChangePwd(res http.ResponseWriter, req *http.Request) {
	// Grab the username/password from the submitted post form
	passwordOld := req.FormValue("passwordO")
	passwordA := req.FormValue("passwordA")
	passwordB := req.FormValue("passwordB")
	username := getUsername(req)

	// Grab from the database
	var databaseUsername string
	var databasePassword string

	// Search the database for the username provided
	err := db.QueryRow("SELECT Username, Password FROM Users WHERE Username=?", username).Scan(&databaseUsername, &databasePassword)
	// If not then redirect to the login page
	if err != nil {
		http.Redirect(res, req, "/login", http.StatusFound)
		return
	}

	// Validate the password
	err = bcrypt.CompareHashAndPassword([]byte(databasePassword), []byte(passwordOld))
	// If wrong password redirect to the login
	if err != nil {
		http.Redirect(res, req, "/dashboard/user/settings", http.StatusFound)
		return
	}

	if (passwordA == passwordB) && (passwordA != "") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordA), bcrypt.DefaultCost)

		_, err = db.Exec("UPDATE Users SET Password=? where Username=?", hashedPassword, username)
		if err != nil {
			databaseError(res, req, err)
			return
		}
	}

	if (passwordA != passwordB) || (passwordA == "") {
		logicError(res, req, "Passwords do not match")
		return
	}

	http.Redirect(res, req, "/dashboard/user/settings", http.StatusFound)
}

func showNewUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	type Entry struct {
		RoleID   int
		RoleName string
	}

	var Roles []Entry
	var Role Entry

	rows, err := db.Query("SELECT RoleID, RoleName FROM Roles")
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Role.RoleID, &Role.RoleName)
		if err != nil {
			databaseError(res, req, err)
			return
		}
		Roles = append(Roles, Role)
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
		"Roles":             Roles,
	}

	t := createHTML("/user/new.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitNewUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	username := req.FormValue("username")
	firstname := req.FormValue("firstname")
	lastname := req.FormValue("lastname")
	email := req.FormValue("email")
	role := req.FormValue("role")
	password := req.FormValue("passwordA")

	var user string

	err := db.QueryRow("SELECT Username FROM Users WHERE Username=?", username).Scan(&user)
	switch {
	case err == sql.ErrNoRows: // Username is available
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}

		_, err = db.Exec("INSERT INTO Users(Username, Password, EmailAddress, Role, FirstName, LastName ) VALUES(?, ?, ?, ?, ?, ?)", username, hashedPassword, email, role, firstname, lastname)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		http.Redirect(res, req, "/dashboard/user/settings", http.StatusFound)
		return

	case err != nil:
		databaseError(res, req, err)
		return

	default:
		logicError(res, req, "Username in use")
		return
	}
}

func showListUsers(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	type Entry struct {
		ID       int
		Username string
		Access   string
		Role     int
	}

	var Users []Entry
	var User Entry

	rows, err := db.Query("SELECT ID, Username, Role FROM Users")
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&User.ID, &User.Username, &User.Role)
		if err != nil {
			databaseError(res, req, err)
			return
		}
		err = db.QueryRow("SELECT RoleName FROM Roles WHERE RoleID=?", User.Role).Scan(&User.Access)
		Users = append(Users, User)
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessEditUser":    (Access & EDITUSER),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
		"Users":             Users,
	}

	t := createHTML("/user/list.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showEditUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	EditUserID := req.FormValue("ID")
	EditUserName := ""
	EditUserEmail := ""
	EditUserRole := 0

	type Entry struct {
		RoleID   int
		RoleName string
	}

	var Roles []Entry
	var Role Entry

	rows, err := db.Query("SELECT RoleID, RoleName FROM Roles")
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Role.RoleID, &Role.RoleName)
		if err != nil {
			databaseError(res, req, err)
			return
		}
		Roles = append(Roles, Role)
	}

	err = db.QueryRow("SELECT Username, EmailAddress FROM Users WHERE ID=?", EditUserID).Scan(&EditUserName, &EditUserEmail)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewUser":     (Access & NEWUSER),
		"AccessEditUser":    (Access & EDITUSER),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"EditUserID":        EditUserID,
		"EditUserName":      EditUserName,
		"EditUserEmail":     EditUserEmail,
		"EditUserRole":      EditUserRole,
		csrf.TemplateTag:    csrf.TemplateField(req),
		"Roles":             Roles,
	}

	t := createHTML("/user/edit.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitEditUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	EditUserID := req.FormValue("ID")
	EditUserEmail := req.FormValue("email")
	EditUserRole := req.FormValue("role")

	_, err = db.Exec("UPDATE Users SET EmailAddress=?, Role=? WHERE ID=?", EditUserEmail, EditUserRole, EditUserID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	http.Redirect(res, req, "/dashboard/users", http.StatusFound)
	return
}

func submitDeleteUser(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITUSER) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	EditUserID := req.FormValue("ID")

	_, err = db.Exec("DELETE from Users WHERE ID=?", EditUserID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	http.Redirect(res, req, "/dashboard/users", http.StatusFound)
	return
}

func showChangeEmail(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/user/change_email.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitChangeEmail(res http.ResponseWriter, req *http.Request) {
	EmailA := req.FormValue("EmailA")
	EmailB := req.FormValue("EmailB")
	username := getUsername(req)

	if (EmailA == EmailB) && (EmailA != "") {
		_, err = db.Exec("UPDATE Users SET EmailAddress=? WHERE Username=?", EmailA, username)
		if err != nil {
			databaseError(res, req, err)
			return
		}
	}

	if (EmailA != EmailB) && (EmailA == "") {
		logicError(res, req, "Email address dont match")
		return
	}

	http.Redirect(res, req, "/dashboard/user/settings", http.StatusFound)
}
