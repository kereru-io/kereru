package main

import _ "github.com/go-sql-driver/mysql"
import "github.com/gorilla/csrf"
import "log"
import "net/http"

//Access control magic numbers
const (
	NEWUSER      int = 1
	EDITUSER     int = 2
	RBAC         int = 4
	NEWTWEET     int = 8
	EDITTWEET    int = 16
	REVIEWTWEET  int = 32
	PUBLISHTWEET int = 64
	FLAGTWEET    int = 128
	DELETETWEET  int = 256
	NEWIMAGE     int = 512
	EDITIMAGE    int = 1024
	DELETEIMAGE  int = 2048
	AUDITTWEET   int = 4096
	NEWVIDEO     int = 8192
	EDITVIDEO    int = 16384
	DELETEVIDEO  int = 32768
)

func submitDeleteRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	if (Access & RBAC) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	DeleteID := req.FormValue("ID")

	_, err = db.Exec("DELETE from Roles WHERE RoleID=?", DeleteID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	http.Redirect(res, req, "/dashboard/rbac", http.StatusFound)
	return
}

func showNewRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	if (Access & RBAC) == 0 {
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

	t := createHTML("/rbac/new.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func submitNewRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & RBAC) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	RoleName := req.FormValue("RoleName")
	DatabaseRole := ""
	err := db.QueryRow("SELECT RoleName FROM Roles WHERE RoleName=?", RoleName).Scan(&DatabaseRole)
	if RoleName == DatabaseRole {
		logicError(res, req, "Role name is in use")
		return
	}

	NewAccess := 0
	if req.FormValue("NewUser") == "Y" {
		NewAccess = NewAccess + NEWUSER
	}
	if req.FormValue("EditUser") == "Y" {
		NewAccess = NewAccess + EDITUSER
	}
	if req.FormValue("RBAC") == "Y" {
		NewAccess = NewAccess + RBAC
	}
	if req.FormValue("NewTweet") == "Y" {
		NewAccess = NewAccess + NEWTWEET
	}
	if req.FormValue("EditTweet") == "Y" {
		NewAccess = NewAccess + EDITTWEET
	}
	if req.FormValue("ReviewTweet") == "Y" {
		NewAccess = NewAccess + REVIEWTWEET
	}
	if req.FormValue("PublishTweet") == "Y" {
		NewAccess = NewAccess + PUBLISHTWEET
	}
	if req.FormValue("FlagTweet") == "Y" {
		NewAccess = NewAccess + FLAGTWEET
	}
	if req.FormValue("DeleteTweet") == "Y" {
		NewAccess = NewAccess + DELETETWEET
	}
	if req.FormValue("ImageUpload") == "Y" {
		NewAccess = NewAccess + NEWIMAGE
	}
	if req.FormValue("ImageEdit") == "Y" {
		NewAccess = NewAccess + EDITIMAGE
	}
	if req.FormValue("ImageDelete") == "Y" {
		NewAccess = NewAccess + DELETEIMAGE
	}
	if req.FormValue("TweetAudit") == "Y" {
		NewAccess = NewAccess + AUDITTWEET
	}
	if req.FormValue("VideoUpload") == "Y" {
		NewAccess = NewAccess + NEWVIDEO
	}
	if req.FormValue("VideoEdit") == "Y" {
		NewAccess = NewAccess + EDITVIDEO
	}
	if req.FormValue("VideoDelete") == "Y" {
		NewAccess = NewAccess + DELETEVIDEO
	}

	_, err = db.Exec("INSERT INTO Roles(RoleName, Access) VALUES(?, ?);", RoleName, NewAccess)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/rbac", http.StatusFound)
}

func showListRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & RBAC) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	type Entry struct {
		ID     int
		Name   string
		Access string
	}

	var Roles []Entry
	var Role Entry

	var UserAccess int

	Search := "SELECT RoleID, RoleName, Access FROM Roles"

	rows, err := db.Query(Search)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Role.ID, &Role.Name, &UserAccess)
		if err != nil {
			databaseError(res, req, err)
			return
		}
		Role.Access = getRoleAsString(UserAccess)
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
	t := createHTML("/rbac/list.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func showEditRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & RBAC) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	RoleID := req.FormValue("ID")
	RoleName := ""
	RoleAccess := 0

	err = db.QueryRow("SELECT RoleName, Access from Roles WHERE RoleID=?", RoleID).Scan(&RoleName, &RoleAccess)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"RoleID":            RoleID,
		"Rolename":          RoleName,
		"NewUser":           (RoleAccess & NEWUSER),
		"EditUser":          (RoleAccess & EDITUSER),
		"RBAC":              (RoleAccess & RBAC),
		"NewTweet":          (RoleAccess & NEWTWEET),
		"EditTweet":         (RoleAccess & EDITTWEET),
		"ReviewTweet":       (RoleAccess & REVIEWTWEET),
		"PublishTweet":      (RoleAccess & PUBLISHTWEET),
		"FlagTweet":         (RoleAccess & FLAGTWEET),
		"DeleteTweet":       (RoleAccess & DELETETWEET),
		"ImageUpload":       (RoleAccess & NEWIMAGE),
		"ImageEdit":         (RoleAccess & EDITIMAGE),
		"ImageDelete":       (RoleAccess & DELETEIMAGE),
		"TweetAudit":        (RoleAccess & AUDITTWEET),
		"VideoUpload":       (RoleAccess & NEWVIDEO),
		"VideoEdit":         (RoleAccess & EDITVIDEO),
		"VideoDelete":       (RoleAccess & DELETEVIDEO),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}

	t := createHTML("/rbac/edit.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func submitEditRBAC(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & RBAC) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	RoleName := req.FormValue("RoleName")
	RoleID := req.FormValue("RoleID")

	access := 0
	if req.FormValue("NewUser") == "Y" {
		access = access + NEWUSER
	}
	if req.FormValue("EditUser") == "Y" {
		access = access + EDITUSER
	}
	if req.FormValue("RBAC") == "Y" {
		access = access + RBAC
	}
	if req.FormValue("NewTweet") == "Y" {
		access = access + NEWTWEET
	}
	if req.FormValue("EditTweet") == "Y" {
		access = access + EDITTWEET
	}
	if req.FormValue("ReviewTweet") == "Y" {
		access = access + REVIEWTWEET
	}
	if req.FormValue("PublishTweet") == "Y" {
		access = access + PUBLISHTWEET
	}
	if req.FormValue("FlagTweet") == "Y" {
		access = access + FLAGTWEET
	}
	if req.FormValue("DeleteTweet") == "Y" {
		access = access + DELETETWEET
	}
	if req.FormValue("ImageUpload") == "Y" {
		access = access + NEWIMAGE
	}
	if req.FormValue("ImageEdit") == "Y" {
		access = access + EDITIMAGE
	}
	if req.FormValue("ImageDelete") == "Y" {
		access = access + DELETEIMAGE
	}
	if req.FormValue("TweetAudit") == "Y" {
		access = access + AUDITTWEET
	}
	if req.FormValue("VideoUpload") == "Y" {
		access = access + NEWVIDEO
	}
	if req.FormValue("VideoEdit") == "Y" {
		access = access + EDITVIDEO
	}
	if req.FormValue("VideoDelete") == "Y" {
		access = access + DELETEVIDEO
	}

	_, err = db.Exec("UPDATE Roles set RoleName=?, Access=? WHERE RoleID=?", RoleName, access, RoleID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/rbac", http.StatusFound)
}
