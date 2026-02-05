package internal

import (
	"embed"
	"fmt"
	"github.com/TUM-Dev/meldeplattform/pkg/mail"
	"github.com/TUM-Dev/meldeplattform/pkg/middleware"
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
	"html/template"
	"io"
	"log"
	"net/http"
	gomail "net/mail"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// parseUintParam safely parses a URL parameter as a uint, returning 0 if invalid
func parseUintParam(c *gin.Context, param string) (uint, error) {
	val := c.Param(param)
	id, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be a positive integer", param)
	}
	return uint(id), nil
}

// allowedFileExtensions contains file extensions that are allowed for upload
var allowedFileExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".txt": true, ".csv": true, ".odt": true, ".ods": true, ".rtf": true,
	".zip": true, ".tar": true, ".gz": true, ".7z": true,
	".mp4": true, ".webm": true, ".mp3": true, ".wav": true,
}

// maxFileSize is the maximum allowed file size (10 MB)
const maxFileSize = 10 << 20

// sanitizeFilename removes path traversal attempts and ensures a safe base filename
func sanitizeFilename(filename string) string {
	// Get only the base name (removes any path components such as "..", "/" and "\")
	filename = filepath.Base(filename)
	// If filename is empty or just dots after sanitization, generate a random one
	if filename == "" || filename == "." {
		filename = uuid.New().String()
	}
	return filename
}

// isAllowedFileExtension checks if the file extension is in the allowlist
func isAllowedFileExtension(ext string) bool {
	return allowedFileExtensions[ext]
}

//go:embed web/templates
var templates embed.FS

//go:embed web/dist
var static embed.FS

func (a *App) initRoutes() {
	funcs := map[string]interface{}{
		"getByIndex": func(els []model.Topic, i *int) model.Topic {
			if i == nil || *i < 0 || *i >= len(els) {
				return model.Topic{}
			}
			return els[*i]
		},
	}
	a.template = template.Must(template.New("base").Funcs(funcs).ParseFS(templates, "web/templates/*.gohtml"))

	middleware.ConfigSaml(a.engine, a.config.Saml)

	a.engine.Use(middleware.InitI18n)
	a.engine.Use(middleware.InitTemplateBase(a.i18n, a.config, a.db))

	a.engine.GET("/", a.indexRoute)
	a.engine.GET("/imprint", a.infoRoute("imprint"))
	a.engine.GET("/privacy", a.infoRoute("privacy"))
	a.engine.GET("/setLang", a.setLang)
	a.engine.GET("/form/:topicID", a.formRoute)
	a.engine.POST("/submit", a.submitRoute)
	a.engine.GET("/file/:name", a.getFile)
	a.engine.GET("/report", a.reportRoute)
	a.engine.POST("/report", a.replyRoute)
	a.engine.StaticFS("/static", http.FS(static))

	createTopicGroup := a.engine.Group("/newTopic/:topicID").Use(middleware.AuthAdminOfTopic(a.db))
	createTopicGroup.GET("", a.newTopicRoute)

	reportsOfTopicGroup := a.engine.Group("/reports/:topicID").Use(middleware.AuthAdminOfTopic(a.db))
	reportsOfTopicGroup.GET("", a.reportsOfTopicRoute)

	adminOfTopicRoutes := a.engine.Group("/api/topic/:topicID").Use(middleware.AuthAdminOfTopic(a.db))

	adminOfTopicRoutes.POST("/report/:reportID/status", a.setStatus)
	adminOfTopicRoutes.GET("", a.getTopic)
	adminOfTopicRoutes.POST("", a.upsertTopic)

}

func (a *App) infoRoute(page string) func(c *gin.Context) {
	var content template.HTML
	switch page {
	case "imprint":
		content = a.config.GetImprint()
	case "privacy":
		content = a.config.GetPrivacy()
	}

	return func(c *gin.Context) {
		err := a.template.ExecuteTemplate(c.Writer, "info.gohtml", model.InfoPage{
			Base:    c.MustGet("base").(model.Base),
			Content: content,
		})
		if err != nil {
			log.Println("template error:", err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}

func (a *App) setStatus(c *gin.Context) {
	var req struct {
		S string `json:"s"`
	}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid request")
		return
	}
	reportID, err := parseUintParam(c, "reportID")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid report ID")
		return
	}
	topicID, err := parseUintParam(c, "topicID")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid topic ID")
		return
	}
	var r model.Report
	err = a.db.First(&r, reportID).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "report not found")
		return
	}
	if r.TopicID != topicID {
		c.AbortWithStatusJSON(http.StatusBadRequest, "report not found")
		return
	}
	newStatus := model.ReportStateOpen
	switch req.S {
	case "open":
		newStatus = model.ReportStateOpen
	case "close":
		newStatus = model.ReportStateDone
	case "spam":
		newStatus = model.ReportStateSpam
	default:
		return
	}
	if err := a.db.Model(&model.Report{}).Where("id = ?", r.ID).Update("state", newStatus).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "failed to update status")
		return
	}
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml",
		model.Index{
			Base:  c.MustGet("base").(model.Base),
			Topic: nil,
		})
	if err != nil {
		log.Println("template error:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

// xsrfTokens is a map of xsrf tokens to the time they were created
var xsrfTokens = map[string]time.Time{}
var xsrfMutex sync.RWMutex

func init() {
	// Start a goroutine to periodically clean up expired XSRF tokens
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			cleanupExpiredXSRFTokens()
		}
	}()
}

// cleanupExpiredXSRFTokens removes tokens older than 30 minutes
func cleanupExpiredXSRFTokens() {
	xsrfMutex.Lock()
	defer xsrfMutex.Unlock()
	cutoff := time.Now().Add(-30 * time.Minute)
	for token, created := range xsrfTokens {
		if created.Before(cutoff) {
			delete(xsrfTokens, token)
		}
	}
}

func (a *App) formRoute(c *gin.Context) {
	base := c.MustGet("base").(model.Base)
	topicID, err := parseUintParam(c, "topicID")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid topic ID")
		return
	}
	var topic model.Topic
	if err := a.db.Preload(clause.Associations).First(&topic, topicID).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	token := uuid.New().String()
	xsrfMutex.Lock()
	xsrfTokens[token] = time.Now()
	xsrfMutex.Unlock()

	if err = a.template.ExecuteTemplate(c.Writer, "index.gohtml", model.Index{
		Topic: &topic,
		Base:  base,
		Token: token,
	}); err != nil {
		log.Println("template error:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (a *App) submitRoute(c *gin.Context) {
	message := ""
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid form data")
		return
	}

	token := c.PostForm("token")
	xsrfMutex.Lock()
	t, ok := xsrfTokens[token]
	if ok {
		delete(xsrfTokens, token)
	}
	xsrfMutex.Unlock()
	if !ok || time.Now().After(t.Add(time.Minute*30)) {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid token")
		return
	}

	topicIDStr := c.PostForm("topic")
	topicID, err := strconv.ParseUint(topicIDStr, 10, 32)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid topic ID")
		return
	}
	var topic model.Topic
	err = a.db.Preload(clause.Associations).First(&topic, uint(topicID)).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "topic not found")
		return
	}
	for _, field := range topic.Fields {
		message += "\n**" + field.Name.En + "**\n" // todo get language from somewhere

		// handle text-ish fields
		if field.Type != "file" && field.Type != "files" {
			fieldResp := c.PostForm(fmt.Sprintf("%d", field.ID))
			if fieldResp == "" && field.Required {
				c.AbortWithStatusJSON(http.StatusBadRequest, "required field not provided")
				return
			}
			message += fieldResp + "\n"
			continue
		}

		// handle file fields
		multipartFile, ok := c.Request.MultipartForm.File[fmt.Sprintf("%d", field.ID)]
		if !ok && field.Required {
			c.AbortWithStatusJSON(http.StatusBadRequest, "required field not provided 1")
			return
		}
		if ok {
			for _, f := range multipartFile {
				// Validate file size
				if f.Size > maxFileSize {
					c.AbortWithStatusJSON(http.StatusBadRequest, "file too large (max 10MB)")
					return
				}
				// Sanitize filename to prevent path traversal
				sanitizedName := sanitizeFilename(f.Filename)
				// Extract and validate file extension
				ext := strings.ToLower(filepath.Ext(sanitizedName))
				if !isAllowedFileExtension(ext) {
					c.AbortWithStatusJSON(http.StatusBadRequest, "file type not allowed")
					return
				}
				open, err := f.Open()
				if err != nil {
					log.Println("failed to open uploaded file:", err)
					continue
				}
				// Use UUID for storage filename to prevent conflicts and enumeration
				storageFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
				filePath := path.Join(a.config.FileDir, storageFilename)
				file, err := os.Create(filePath)
				if err != nil {
					log.Println("failed to create file:", err)
					_ = open.Close()
					continue
				}
				_, err = io.Copy(file, open)
				_ = file.Close()
				_ = open.Close()
				if err != nil {
					log.Println("failed to write file:", err)
					_ = os.Remove(filePath)
					continue
				}
				dbFile := model.File{
					Location: filePath,
					Name:     sanitizedName,
				}
				if err := a.db.Create(&dbFile).Error; err != nil {
					log.Println("failed to save file record:", err)
					_ = os.Remove(filePath)
					continue
				}
				message += "[" + dbFile.Name + "](" + a.config.URL + "/file/" + url.QueryEscape(dbFile.Name) + "?id=" + dbFile.UUID + ")"
			}
		}
	}
	dbMsg := model.Message{
		Content: message,
	}
	var email *string
	if emailStr := c.PostForm("email"); emailStr != "" {
		if _, err := gomail.ParseAddress(emailStr); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, "invalid email format")
			return
		}
		email = &emailStr
	}
	dbReport := model.Report{
		TopicID:  topic.ID,
		Messages: []model.Message{dbMsg},
		State:    model.ReportStateOpen,
		Creator:  email,
	}
	err = a.db.Create(&dbReport).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "can't create report")
	}
	err = mail.SendMail(
		a.config.Mail.User,
		a.config.Mail.Password,
		a.config.Mail.SMTPServer, a.config.Mail.SMTPPort,
		a.config.Mail.FromName, a.config.Mail.From, topic.Email,
		fmt.Sprintf("[%s]: report #%d opened", topic.Name.En, dbReport.ID),
		"Hi, there is a new report regarding "+topic.Name.En+":\n\n"+string(dbMsg.GetBody())+
			"\n\nYou can reply to it <a href=\""+a.config.URL+"/report?administratorToken="+dbReport.AdministratorToken+"\">here</a>.")
	if err != nil {
		log.Println(err)
	}
	c.Redirect(http.StatusFound, "/report?reporterToken="+dbReport.ReporterToken)
}

func (a *App) reportRoute(c *gin.Context) {
	d := model.ReportPage{
		Base:   c.MustGet("base").(model.Base),
		Report: &model.Report{},
	}

	reporterToken := c.Query("reporterToken")
	administratorToken := c.Query("administratorToken")
	if administratorToken != "" {
		d.IsAdministrator = true
		if err := a.db.Where("administrator_token = ?", administratorToken).
			Preload("Messages").
			Find(d.Report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else if reporterToken != "" {
		d.IsAdministrator = false
		if err := a.db.Where("reporter_token = ?", reporterToken).
			Preload("Messages").
			Find(d.Report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	err := a.template.ExecuteTemplate(c.Writer, "report.gohtml", d)
	if err != nil {
		log.Println("template error:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (a *App) getFile(c *gin.Context) {
	fileID := c.Query("id")
	if fileID == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	res := model.File{}
	err := a.db.Where("uuid = ?", fileID).Find(&res).Error
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	// Validate that file location is within allowed directory (defense-in-depth)
	absLocation, err := filepath.Abs(res.Location)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	absFileDir, err := filepath.Abs(a.config.FileDir)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absLocation, absFileDir) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.FileAttachment(res.Location, res.Name)
}

func (a *App) replyRoute(c *gin.Context) {
	reporterToken := c.Query("reporterToken")
	administratorToken := c.Query("administratorToken")
	isAdmin := false
	var report = model.Report{}
	if reporterToken != "" {
		if err := a.db.Where("reporter_token = ?", reporterToken).Find(&report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else if administratorToken != "" {
		isAdmin = true
		if err := a.db.Where("administrator_token = ?", administratorToken).Find(&report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else {
		c.AbortWithStatusJSON(http.StatusBadRequest, "no token provided")
		return
	}
	var topic model.Topic
	if a.db.Find(&topic, report.TopicID).Error != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "can't find topic")
		return
	}
	reply := c.PostForm("reply")
	if len(reply) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, "empty reply")
		return
	}
	if err := a.db.Create(&model.Message{
		Content:  reply,
		ReportID: report.ID,
		IsAdmin:  isAdmin,
	}).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "failed to create message")
		return
	}
	err := mail.SendMail(
		a.config.Mail.User, a.config.Mail.Password,
		a.config.Mail.SMTPServer, a.config.Mail.SMTPPort,
		a.config.Mail.FromName, a.config.Mail.From, topic.Email,
		fmt.Sprintf("[%s]: report #%d updated", topic.Name.En, report.ID),
		"Hi, there is a new message regarding "+topic.Name.En+":\n\n"+c.PostForm("reply")+"\n\nYou can reply to it <a href=\""+a.config.URL+"/report?administratorToken="+report.AdministratorToken+"\">here</a>.")
	if err != nil {
		log.Println(err)
	}
	if isAdmin {
		if report.Creator != nil {
			// Validate stored email before sending (defense-in-depth)
			if _, err := gomail.ParseAddress(*report.Creator); err != nil {
				log.Println("invalid creator email:", err)
			} else {
				err := mail.SendMail(
					a.config.Mail.User, a.config.Mail.Password,
					a.config.Mail.SMTPServer, a.config.Mail.SMTPPort,
					a.config.Mail.FromName, a.config.Mail.From, *report.Creator,
					fmt.Sprintf("[%s]: report #%d updated", topic.Name.En, report.ID),
					"Hi, there is a new message regarding "+topic.Name.En+":\n\n"+c.PostForm("reply")+"\n\nYou can reply to it <a href=\""+a.config.URL+"/report?reporterToken="+report.ReporterToken+"\">here</a>.")
				if err != nil {
					log.Println(err)
				}
			}
		}
		c.Redirect(http.StatusFound, "/report?administratorToken="+administratorToken)
	} else {
		c.Redirect(http.StatusFound, "/report?reporterToken="+reporterToken)
	}
}

func (a *App) newTopicRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer,
		"newTopic.gohtml",
		model.NewTopicPage{Base: c.MustGet("base").(model.Base)},
	)
	if err != nil {
		log.Println("template error:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (a *App) getTopic(c *gin.Context) {
	topicID, err := parseUintParam(c, "topicID")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid topic ID")
		return
	}
	if topicID == 0 {
		c.JSON(http.StatusOK, model.Topic{Fields: []model.Field{}, Admins: []model.Admin{}})
		return
	}
	var topic model.Topic
	if err := a.db.Preload(clause.Associations).First(&topic, topicID).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, topic)
}

func (a *App) upsertTopic(c *gin.Context) {
	var r model.Topic
	err := c.BindJSON(&r)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return
	}
	if len(r.Fields) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, "Please provide at least one question")
		return
	}
	var fieldIDs []uint
	for _, field := range r.Fields {
		fieldIDs = append(fieldIDs, field.ID)
	}
	a.db.Table("fields").Where("topic_id = ? and id not in ?", r.ID, fieldIDs).Delete("")
	a.db.Unscoped().Table("topic_admins").Where("topic_id = ?", r.ID).Delete("")
	var cleanAdmins []model.Admin
	for _, admin := range r.Admins {
		if admin.UserID != "" {
			cleanAdmins = append(cleanAdmins, admin)
		}
	}
	r.Admins = cleanAdmins
	err = a.db.Save(&r).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
	if err := a.db.Save(&r.Fields).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}
}

func (a *App) reportsOfTopicRoute(c *gin.Context) {
	topicID, err := parseUintParam(c, "topicID")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "invalid topic ID")
		return
	}
	var topic model.Topic
	if err := a.db.Preload(clause.Associations).First(&topic, topicID).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	var reports []model.Report
	a.db.Preload(clause.Associations).Where("topic_id = ?", topic.ID).Find(&reports)

	err = a.template.ExecuteTemplate(c.Writer, "reportsOfTopic.gohtml", model.ReportsOfTopicPage{
		Base:    c.MustGet("base").(model.Base),
		Topic:   &topic,
		Reports: reports,
	})
	if err != nil {
		log.Println("template error:", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
