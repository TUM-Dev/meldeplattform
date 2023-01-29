package internal

import (
	"github.com/TUM-Dev/meldeplattform/pkg/middleware"
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"github.com/gin-gonic/gin"

	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
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

	a.engine.Use(middleware.InitI18n)
	a.engine.Use(middleware.InitTemplateBase(a.i18n, a.config))

	a.engine.GET("/", a.indexRoute)
	a.engine.GET("/setLang", a.setLang)
	a.engine.GET("/form/:topicID", a.formRoute)
	a.engine.POST("/submit", a.submitRoute)
	a.engine.GET("/file/:name", a.getFile)
	a.engine.GET("/report", a.reportRoute)
	a.engine.POST("/report", a.replyRoute)
	a.engine.StaticFS("/static", http.FS(static))
	a.engine.POST("/setStatus/:administratorToken", a.setStatus)
}

func (a *App) setStatus(c *gin.Context) {
	administratorToken := c.Param("administratorToken")
	if administratorToken == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	report := model.Report{}
	if err := a.db.Where("administrator_token = ?", administratorToken).
		Find(&report).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	var r struct{ Status string }
	err := c.BindJSON(&r)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	switch r.Status {
	case "open":
		report.State = model.ReportStateOpen
	case "done":
		report.State = model.ReportStateDone
	}
	if err := a.db.Debug().Save(&report).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml",
		model.Index{
			Base:   c.MustGet("base").(model.Base),
			Topics: a.config.Content.Topics,
			Topic:  nil,
		})
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) formRoute(c *gin.Context) {
	var topic *int
	t := c.Param("topicID")
	if t != "" {
		atoi, err := strconv.Atoi(t)
		if err != nil {
			topic = nil
		} else {
			if atoi >= len(a.config.Content.Topics) || atoi < 0 {
				topic = nil
			} else {
				topic = &atoi
			}
		}
	}

	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", model.Index{
		Topic: topic,
		Base:  c.MustGet("base").(model.Base),
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

	topic := a.config.Content.Topics[0] // todo: get Topic from form
	for i, field := range topic.Fields {
		message += "\n**" + field.Name.En + "**\n" // todo get language from somewhere

		// handle text-ish fields
		if field.Type != "file" {
			fieldResp := c.PostForm(fmt.Sprintf("%d", i))
			if fieldResp == "" && field.Required {
				c.AbortWithStatusJSON(http.StatusBadRequest, "required field not provided")
				return
			}
			message += fieldResp + "\n"
			continue
		}

		// handle file fields
		multipartFile, ok := c.Request.MultipartForm.File[fmt.Sprintf("%d", i)]
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
	dbReport := model.Report{
		Messages: []model.Message{dbMsg},
		State:    model.ReportStateOpen,
	}
	err = a.db.Create(&dbReport).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "can't create report")
	}
	for _, m := range model.GetMessengers(topic) {
		err := m.SendMessage(fmt.Sprintf("Report #%d opened", dbReport.ID), dbMsg, a.config.URL+"/report?administratorToken="+dbReport.AdministratorToken)
		if err != nil {
			log.Println("Can't send message:", err)
		}
	}
	c.Redirect(http.StatusFound, "/report?reporterToken="+dbReport.ReporterToken)
}

type ReportPageData struct {
	Config model.Config
	Report *model.Report

	IsAdministrator bool
}

func (a *App) reportRoute(c *gin.Context) {
	d := ReportPageData{
		Config: a.config,
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
	report := model.Report{}
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
	a.db.Create(&model.Message{
		Content:  c.PostForm("reply"),
		ReportID: report.ID,
		IsAdmin:  isAdmin,
	})
	if isAdmin {
		c.Redirect(http.StatusFound, "/report?administratorToken="+administratorToken)
	} else {
		c.Redirect(http.StatusFound, "/report?reporterToken="+reporterToken)
	}
}
