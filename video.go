package main

import "os"
import _ "github.com/go-sql-driver/mysql"
import "github.com/satori/go.uuid"
import "github.com/gorilla/csrf"
import "io"
import "fmt"
import "strconv"
import "log"
import "net/http"
import "time"
import "os/exec"
import "encoding/json"

func getVidPageCount(res http.ResponseWriter, req *http.Request) {
	TotalVideos := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Videos").Scan(&TotalVideos)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	type Entry struct {
		Totalpages int64
	}

	var Data Entry

	Data.Totalpages = int64(TotalVideos/12) + 1

	var jsonData []byte
	jsonData, err = json.Marshal(Data)
	if err != nil {
		logicError(res, req, err.Error())
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func getOneVid(res http.ResponseWriter, req *http.Request) {
	VideoID := req.FormValue("ID")
	if VideoID == "" {
		VideoID = "0"
	}

	type Entry struct {
		ID   int64
		GUID string
		DESC string
	}

	var Video Entry

	err = db.QueryRow("SELECT ID, GUID, Description FROM Videos WHERE ID=?", VideoID).Scan(&Video.ID, &Video.GUID, Video.DESC)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	var jsonData []byte
	jsonData, err = json.Marshal(Video)
	if err != nil {
		logicError(res, req, err.Error())
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func getListVid(res http.ResponseWriter, req *http.Request) {
	page, err := strconv.Atoi(req.FormValue("page"))

	if page == 0 {
		page = 1
	}

	offset := (page - 1) * 12

	type Entry struct {
		ID   int64
		GUID string
		DESC string
	}

	var Video Entry
	var Videos []Entry

	rows, err := db.Query("SELECT ID, GUID, Description FROM Videos ORDER BY UploadTime DESC LIMIT ?,?", offset, 12)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Video.ID, &Video.GUID, &Video.DESC)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		Videos = append(Videos, Video)
	}

	var jsonData []byte
	jsonData, err = json.Marshal(Videos)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func showNewVideo(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWVIDEO) == 0 {
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

	t := createHTML("/video/new.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func submitNewVideo(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWVIDEO) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	req.ParseMultipartForm(64 << 20) //max file size 64mb

	VideoName := req.FormValue("VideoName")
	Description := req.FormValue("Description")
	VideoNotes := req.FormValue("Notes")
	InputFile, InputHandler, err := req.FormFile("UploadFile")
	if err != nil {
		log.Print("Something went wrong during file upload: %v", err)
		return
	}
	defer InputFile.Close()

	buffer := make([]byte, 512)
	_, err = InputFile.Read(buffer)
	if err != nil {
		logicError(res, req, "File upload failed!")
	}
	InputFile.Seek(0, 0)
	contentType := http.DetectContentType(buffer)
	if contentType != "video/mp4" {
		log.Print(contentType)
		http.Redirect(res, req, "/dashboard/videos/error", http.StatusFound)
		return
	}

	uuidfile := uuid.NewV4()
	guidName := uuidfile.String()
	OutputFile, err := os.OpenFile(config.UploadPath+"/"+guidName, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Print("Something went wrong with writing file to disk: %v", err)
		return
	}
	FileTime := time.Now().Unix()
	_, err = db.Exec("INSERT INTO Videos(GUID, VideoName, Description, UploadTime, FileSize, FileName,Notes,MediaTime) VALUES(?, ?, ?, ?, ?, ?, ?, ?)", guidName, VideoName, Description, FileTime, InputHandler.Size, InputHandler.Filename, VideoNotes, 0)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	defer OutputFile.Close()
	io.Copy(OutputFile, InputFile)

	thumbnailVideo(guidName)

	http.Redirect(res, req, "/dashboard/videos", http.StatusFound)
}

func showListVideos(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	DisplayPagination := getUserDisplayPagination(req)
	VideoIndex, err := strconv.Atoi(req.FormValue("Video"))
	Page, err := strconv.Atoi(req.FormValue("page"))

	type Entry struct {
		ID          int
		GUID        string
		Name        string
		Filename    string
		Description string
		Time        string
	}

	var Videos []Entry
	var Video Entry
	var VideoTime int64

	TotalVideos := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Videos").Scan(&TotalVideos)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	if Page > (TotalVideos/100)+1 {
		Page = 1
	}

	if (Page == 0) && (VideoIndex > 0) {
		Page = getVideoPage(VideoIndex)
	}

	if Page == 0 {
		Page = 1
	}

	Search := "SELECT ID, GUID, VideoName, Description, UploadTime, FileName FROM Videos ORDER BY UploadTime DESC"

	if DisplayPagination == 1 {
		Search = Search + " LIMIT " + strconv.Itoa((Page-1)*100) + ", 100"
	}

	rows, err := db.Query(Search)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&Video.ID, &Video.GUID, &Video.Name, &Video.Description, &VideoTime, &Video.Filename)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		const DateFormat = "2006-01-02 at 15:04:05"
		TimeObj := time.Unix(VideoTime, 0)
		Video.Time = TimeObj.Format(DateFormat)
		Videos = append(Videos, Video)
	}

	PageLast := int(TotalVideos/100) + 1
	PagePre := Page - 1
	PageNext := Page + 1

	if PageNext > PageLast {
		PageNext = PageLast
	}
	if PagePre < 1 {
		PagePre = 1
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"AccessEditVideo":   (Access & EDITVIDEO),
		"AccessDeleteVideo": (Access & DELETEVIDEO),
		"DisplayPagination": DisplayPagination,

		"PagePre":  PagePre,
		"Page":     Page,
		"PageNext": PageNext,
		"PageLast": PageLast,

		csrf.TemplateTag: csrf.TemplateField(req),
		"Videos":         Videos,
	}
	t := createHTML("/video/list.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showEditVideo(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	VideoID := req.FormValue("ID")
	VideoGUID := ""
	VideoDescription := ""
	VideoName := ""
	VideoNotes := ""
	FileSize := 0.0

	err := db.QueryRow("SELECT GUID, VideoName, Description, FileSize, Notes FROM Videos WHERE ID=?", VideoID).Scan(&VideoGUID, &VideoName, &VideoDescription, &FileSize, &VideoNotes)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	FileSizeUnits := ""

	if FileSize > 1024 {
		FileSize = FileSize / 1024.00
		FileSizeUnits = "Kb"
	}

	if FileSize > 1024 {
		FileSize = FileSize / 1024.00
		FileSizeUnits = "Mb"
	}

	FileSizeString := fmt.Sprintf("%.2f", FileSize)
	FileSizeString = FileSizeString + FileSizeUnits

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"AccessEditVideo":   (Access & EDITVIDEO),
		"AccessDeleteVideo": (Access & DELETEVIDEO),
		"GUID":              VideoGUID,
		"VideoName":         VideoName,
		"ID":                VideoID,
		"Description":       VideoDescription,
		"FileSize":          FileSizeString,
		"Notes":             VideoNotes,
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/video/edit.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitDeleteVideo(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & DELETEVIDEO) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	VideoID := req.FormValue("ID")

	VideoGUID := "--"
	err := db.QueryRow("SELECT GUID FROM Videos WHERE ID=?", VideoID).Scan(&VideoGUID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	err = os.Remove(config.UploadPath + "/" + VideoGUID)
	err = os.Remove(config.UploadPath + "/thumbs/" + VideoGUID)

	_, err = db.Exec("Delete from Videos WHERE ID=?", VideoID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/videos", http.StatusFound)
}

func submitEditVideo(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITVIDEO) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	VideoID := req.FormValue("ID")
	VideoName := req.FormValue("VideoName")
	Description := req.FormValue("Description")
	VideoNotes := req.FormValue("Notes")

	_, err = db.Exec("UPDATE Videos SET VideoName=?, Description=? , Notes=? WHERE ID=?", VideoName, Description, VideoNotes, VideoID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/videos?Video="+VideoID+"#Video"+VideoID, http.StatusFound)
}

func thumbnailVideo(VideoGUID string) {
	cmd := exec.Command("ffmpeg", "-i", config.UploadPath+"/"+VideoGUID, "-vframes", "1", "-an", "-s", "100x100", "-ss", "3", "-f", "apng", config.UploadPath+"/thumbs/"+VideoGUID)

	err := cmd.Run()
	if err != nil {
		log.Printf("Video thumbnail error: %v\n", err)
	}
}

func showVideoError(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"error":             getUserLastError(req),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/video/error.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Fatal(err)
	}
}
