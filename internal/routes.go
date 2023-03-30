package internal

import (
	"github.com/TUM-Dev/meldeplattform/pkg/mail"
	"github.com/TUM-Dev/meldeplattform/pkg/middleware"
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

//go:embed web/templates
var templates embed.FS

//go:embed web/dist
var static embed.FS

func (a *App) initRoutes() {
	dir, err := templates.ReadDir(".")
	fmt.Println(err)
	for _, entry := range dir {
		fmt.Println(entry.Name())
	}
	funcs := map[string]interface{}{
		"getByIndex": func(els []model.Topic, i *int) model.Topic {
			if i == nil {
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
			fmt.Println(err)
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
	var r model.Report
	err = a.db.Find(&r, c.Param("reportID")).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, "report not found")
		return
	}
	if fmt.Sprintf("%d", r.TopicID) != c.Param("topicID") {
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
	a.db.Model(&model.Report{}).Where("id = ?", r.ID).Update("state", newStatus)
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml",
		model.Index{
			Base:  c.MustGet("base").(model.Base),
			Topic: nil,
		})
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) formRoute(c *gin.Context) {
	base := c.MustGet("base").(model.Base)
	t := c.Param("topicID")
	var topic model.Topic
	a.db.Preload(clause.Associations).Find(&topic, t)

	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", model.Index{
		Topic: &topic,
		Base:  base,
	})
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) submitRoute(c *gin.Context) {
	message := ""
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		return
	}

	var topic model.Topic
	err = a.db.Preload(clause.Associations).Find(&topic, c.PostForm("topic")).Error
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
				open, err := f.Open()
				if err != nil {
					fmt.Println(err)
					continue
				}
				filePath := path.Join(a.config.FileDir, fmt.Sprintf("%d-%s", time.Now().Unix(), f.Filename))
				file, err := os.Create(filePath)
				if err != nil {
					fmt.Println(err)
					continue
				}
				_, err = io.Copy(file, open)
				if err != nil {
					fmt.Println(err)
					continue
				}
				_ = file.Close()
				_ = open.Close()
				dbFile := model.File{
					Location: filePath,
					Name:     f.Filename,
				}
				a.db.Create(&dbFile)
				message += "[" + dbFile.Name + "](" + a.config.URL + "/file/" + url.QueryEscape(dbFile.Name) + "?id=" + dbFile.UUID + ")"
			}
		}
	}
	dbMsg := model.Message{
		Content: message,
	}
	var email *string
	if c.PostForm("email") != "" {
		emailS := c.PostForm("email")
		email = &emailS
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
		fmt.Println(err)
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
	a.db.Create(&model.Message{
		Content:  c.PostForm("reply"),
		ReportID: report.ID,
		IsAdmin:  isAdmin,
	})
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
		fmt.Println(err)
	}
}

func (a *App) getTopic(c *gin.Context) {
	topicID := c.Param("topicID")
	if topicID == "0" {
		c.JSON(http.StatusOK, model.Topic{Fields: []model.Field{}, Admins: []model.Admin{}})
		return
	}
	var topic model.Topic
	a.db.Preload(clause.Associations).Find(&topic, topicID)
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
}

func (a *App) reportsOfTopicRoute(c *gin.Context) {
	var topic model.Topic
	a.db.Preload(clause.Associations).Find(&topic, c.Param("topicID"))

	var reports []model.Report
	a.db.Preload(clause.Associations).Where("topic_id = ?", topic.ID).Find(&reports)

	err := a.template.ExecuteTemplate(c.Writer, "reportsOfTopic.gohtml", model.ReportsOfTopicPage{
		Base:    c.MustGet("base").(model.Base),
		Topic:   &topic,
		Reports: reports,
	})
	if err != nil {
		fmt.Println(err)
	}
}
