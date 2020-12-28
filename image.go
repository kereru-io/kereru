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
import "image"
import "image/png"
import _ "image/jpeg"
import _ "image/gif"
import "golang.org/x/image/draw"
import "encoding/json"

func getImgPageCount(res http.ResponseWriter, req *http.Request) {
	TotalImages := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Images").Scan(&TotalImages)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	type Entry struct {
		Totalpages int64
	}

	var Data Entry

	Data.Totalpages = int64(TotalImages/12) + 1

	var jsonData []byte
	jsonData, err = json.Marshal(Data)
	if err != nil {
		logicError(res, req, err.Error())
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func getOneImg(res http.ResponseWriter, req *http.Request) {
	ImageID := req.FormValue("ID")
	if ImageID == "" {
		ImageID = "0"
	}

	type Entry struct {
		ID   int64
		GUID string
		DESC string
	}

	var Image Entry

	err = db.QueryRow("SELECT ID, GUID, Description FROM Images WHERE ID=?", ImageID).Scan(&Image.ID, &Image.GUID, &Image.DESC)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	var jsonData []byte
	jsonData, err = json.Marshal(Image)
	if err != nil {
		logicError(res, req, err.Error())
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func getListImg(res http.ResponseWriter, req *http.Request) {
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

	var Image Entry
	var Images []Entry

	rows, err := db.Query("SELECT ID, GUID, Description FROM Images ORDER BY UploadTime DESC LIMIT ?,?", offset, 12)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&Image.ID, &Image.GUID, &Image.DESC)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		Images = append(Images, Image)
	}

	var jsonData []byte
	jsonData, err = json.Marshal(Images)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonData)
}

func showNewImage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWIMAGE) == 0 {
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

	t := createHTML("/image/new.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
	}
}

func submitNewImage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & NEWIMAGE) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	req.ParseMultipartForm(64 << 20) //max file size 64mb

	ImageName := req.FormValue("ImageName")
	Description := req.FormValue("Description")
	ImageNotes := req.FormValue("Notes")
	InputFile, InputHandler, err := req.FormFile("UploadFile")
	if err != nil {
		log.Print("Something went wrong: %s", err)
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
	if (contentType != "image/jpeg") && (contentType != "image/png") && (contentType != "image/gif") {
		log.Print(contentType)
		http.Redirect(res, req, "/dashboard/images/error", http.StatusFound)
		return
	}

	uuidfile := uuid.NewV4()
	guidName := uuidfile.String()
	OutputFile, err := os.OpenFile(config.UploadPath+"/"+guidName, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}

	FileTime := time.Now().Unix()
	_, err = db.Exec("INSERT INTO Images(GUID, ImageName, Description, UploadTime, FileSize, FileName, Notes, MediaTime) VALUES(?, ?, ?, ?, ?, ?, ?, ?)", guidName, ImageName, Description, FileTime, InputHandler.Size, InputHandler.Filename, ImageNotes, 0)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	defer OutputFile.Close()
	io.Copy(OutputFile, InputFile)

	if InputHandler.Size > 5242000 {
		http.Redirect(res, req, "/dashboard/images/resize?GUID="+guidName, http.StatusFound)
	}

	thumbnailImage(guidName)

	http.Redirect(res, req, "/dashboard/images", http.StatusFound)
}

func showListImages(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	DisplayPagination := getUserDisplayPagination(req)
	ImageIndex, err := strconv.Atoi(req.FormValue("Image"))
	Page, err := strconv.Atoi(req.FormValue("page"))

	type Entry struct {
		ID          int
		GUID        string
		Name        string
		Filename    string
		Description string
		Time        string
	}

	var Images []Entry
	var Image Entry
	var ImageTime int64

	TotalImages := 0
	err = db.QueryRow("SELECT COUNT(*) FROM Images").Scan(&TotalImages)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	if Page > (TotalImages/100)+1 {
		Page = 1
	}

	if (Page == 0) && (ImageIndex > 0) {
		Page = getImagePage(ImageIndex)
	}

	if Page == 0 {
		Page = 1
	}

	Search := "SELECT ID, GUID, ImageName, Description, UploadTime, FileName FROM Images ORDER BY UploadTime DESC"

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
		err = rows.Scan(&Image.ID, &Image.GUID, &Image.Name, &Image.Description, &ImageTime, &Image.Filename)
		if err != nil {
			databaseError(res, req, err)
			return
		}

		const DateFormat = "2006-01-02 at 15:04:05"
		TimeObj := time.Unix(ImageTime, 0)
		Image.Time = TimeObj.Format(DateFormat)
		Images = append(Images, Image)
	}

	PageLast := int(TotalImages/100) + 1
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
		"AccessEditImage":   (Access & EDITIMAGE),
		"AccessDeleteImage": (Access & DELETEIMAGE),
		"DisplayPagination": DisplayPagination,

		"PagePre":  PagePre,
		"Page":     Page,
		"PageNext": PageNext,
		"PageLast": PageLast,

		csrf.TemplateTag: csrf.TemplateField(req),
		"Images":         Images,
	}
	t := createHTML("/image/list.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func showEditImage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	ImageID := req.FormValue("ID")
	ImageGUID := ""
	ImageDescription := ""
	ImageName := ""
	ImageNotes := ""
	FileSize := 0.0

	err := db.QueryRow("SELECT GUID, ImageName, Description, FileSize, Notes FROM Images WHERE ID=?", ImageID).Scan(&ImageGUID, &ImageName, &ImageDescription, &FileSize, &ImageNotes)
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
		"AccessEditImage":   (Access & EDITIMAGE),
		"AccessDeleteImage": (Access & DELETEIMAGE),
		"Guid":              ImageGUID,
		"ImageName":         ImageName,
		"ID":                ImageID,
		"Description":       ImageDescription,
		"FileSize":          FileSizeString,
		"Notes":             ImageNotes,
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/image/edit.tmpl")
	err = t.Execute(res, vars)
	if err != nil {
		log.Print("Something went wrong: %s", err)
		return
	}
}

func submitDeleteImage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & DELETEIMAGE) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	ImageID := req.FormValue("ID")

	ImageGUID := "--"
	err := db.QueryRow("SELECT GUID FROM Images WHERE ID=?", ImageID).Scan(&ImageGUID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	err = os.Remove(config.UploadPath + "/" + ImageGUID)
	err = os.Remove(config.UploadPath + "/thumbs/" + ImageGUID)

	_, err = db.Exec("Delete from Images WHERE ID=?", ImageID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/images", http.StatusFound)
}

func submitEditImage(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	if (Access & EDITIMAGE) == 0 {
		http.Redirect(res, req, "/dashboard/home", http.StatusFound)
		return
	}

	ImageID := req.FormValue("ID")
	ImageName := req.FormValue("ImageName")
	Description := req.FormValue("description")
	ImageNotes := req.FormValue("Notes")

	_, err = db.Exec("UPDATE Images SET ImageName=?, Description=? , Notes=? WHERE ID=?", ImageName, Description, ImageNotes, ImageID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	http.Redirect(res, req, "/dashboard/images?Image="+ImageID+"#Image"+ImageID, http.StatusFound)
}

func thumbnailImage(ImageGUID string) {
	src := openImage(config.UploadPath + "/" + ImageGUID)
	Size := image.Rect(0, 0, 100, 100)
	dst := image.NewRGBA(Size)

	draw.ApproxBiLinear.Scale(dst, Size, src, src.Bounds(), draw.Over, nil)

	dstFile, err := os.Create(config.UploadPath + "/thumbs/" + ImageGUID)
	if err != nil {
		log.Printf("Image Thumbnail Error: %v\n", err)
	}

	err = png.Encode(dstFile, dst)
	dstFile.Close()
	if err != nil {
		log.Printf("Image Thumbnail Error: %v\n", err)
	}
}

func openImage(FileName string) image.Image {
	FileHandel, err := os.Open(FileName)
	if err != nil {
		log.Printf("Image Open Error: %v\n", err)
	}
	defer FileHandel.Close()
	img, _, err := image.Decode(FileHandel)
	if err != nil {
		log.Printf("Image Open error: %v\n", err)
	}
	return img
}

func showImageError(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"error":             getUserLastError(req),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/image/error.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Fatal(err)
	}
}

func showImageResized(res http.ResponseWriter, req *http.Request) {
	Access := getUserAccess(req)
	ImageGUID := req.FormValue("GUID")
	OrignalFileGUID := ""
	err = db.QueryRow("SELECT GUID FROM Images WHERE GUID=?", ImageGUID).Scan(&OrignalFileGUID)
	if err != nil {
		databaseError(res, req, err)
		return
	}
	if OrignalFileGUID == "" {
		http.Redirect(res, req, "/dashboard/images", http.StatusFound)
		return
	}

	vars := map[string]interface{}{
		"UserID":            getUsername(req),
		"AccessNewTweet":    (Access & NEWTWEET),
		"AccessUploadImage": (Access & NEWIMAGE),
		"AccessUploadVideo": (Access & NEWVIDEO),
		"GUID":              OrignalFileGUID,
		"error":             getUserLastError(req),
		csrf.TemplateTag:    csrf.TemplateField(req),
	}
	t := createHTML("/image/resize.tmpl")
	err := t.Execute(res, vars)
	if err != nil {
		log.Fatal(err)
	}
}

func submitImageResized(res http.ResponseWriter, req *http.Request) {
	ImageGUID := req.FormValue("GUID")
	OrignalFileGUID := ""
	err = db.QueryRow("SELECT GUID FROM Images WHERE GUID=?", ImageGUID).Scan(&OrignalFileGUID)
	if OrignalFileGUID == "" {
		http.Redirect(res, req, "/dashboard/images", http.StatusFound)
		return
	}
	Scale := 0.90
	var NewSize int64
	src := openImage(config.UploadPath + "/" + OrignalFileGUID)
	OrignalX := float64(src.Bounds().Max.X)
	OrignalY := float64(src.Bounds().Max.Y)
	uuidfile := uuid.NewV4()
	NewFileGUID := uuidfile.String()

	for i := 0; i < 20; i++ {
		Size := image.Rect(0, 0, int(OrignalX*Scale), int(OrignalY*Scale))
		dst := image.NewRGBA(Size)
		draw.ApproxBiLinear.Scale(dst, Size, src, src.Bounds(), draw.Over, nil)
		dstFile, err := os.Create(config.UploadPath + "/" + NewFileGUID)
		if err != nil {
			log.Printf("Image Resize Error: %v\n", err)
		}

		err = png.Encode(dstFile, dst)
		if err != nil {
			log.Printf("Image Resize Error: %v\n", err)
		}

		dstFile.Close()

		fileHandel, err := os.Stat(config.UploadPath + "/" + NewFileGUID)
		if err != nil {
			log.Printf("Image Resize Error: %v\n", err)
		}

		NewSize = fileHandel.Size()

		if NewSize < 5242000 {
			log.Printf("New Image Scaled to: ", Scale)
			break
		}
		Scale = Scale - 0.10
	}

	_, err = db.Exec("UPDATE Images SET GUID=?,FileSize=? WHERE GUID=?", NewFileGUID, NewSize, OrignalFileGUID)
	if err != nil {
		databaseError(res, req, err)
		return
	}

	thumbnailImage(NewFileGUID)
	http.Redirect(res, req, "/dashboard/images", http.StatusFound)
}
