package internal

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

//go:embed templates
var templates embed.FS

//go:embed css
var static embed.FS

func (a *App) initRoutes() {
	dir, err := templates.ReadDir("/")
	fmt.Println(dir, err)
	funcs := map[string]interface{}{
		"getByIndex": func(els []topic, i *int) topic {
			if i == nil {
				return topic{}
			}
			return els[*i]
		},
	}
	a.template = template.Must(template.New("base").Funcs(funcs).ParseFS(templates, "templates/*.gohtml"))
	a.engine.GET("/", a.indexRoute)
	a.engine.GET("/form/:topicID", a.formRoute)
	a.engine.POST("/submit", a.submitRoute)
	a.engine.GET("/file/:name", a.getFile)
	a.engine.GET("/report", a.reportRoute)
	a.engine.POST("/report", a.replyRoute)
	a.engine.StaticFS("/static", http.FS(static))
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", struct {
		Topic *int
		config
	}{nil, a.config})
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) formRoute(c *gin.Context) {
	var topic *int
	t := c.GetInt("topicID")
	if t < 0 || t >= len(a.config.Content.Topics) {
		topic = nil
	} else {
		topic = intPtr(t)
	}
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", struct {
		Topic *int
		config
	}{topic, a.config})
	if err != nil {
		fmt.Println(err)
	}
}

func intPtr(i int) *int {
	return &i
}

func (a *App) submitRoute(c *gin.Context) {
	message := ""
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		return
	}

	for i, field := range a.config.Content.Topics[0].Fields {
		message += "\n**" + field.Name + "**\n"

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
				dbFile := File{
					Location: filePath,
					Name:     f.Filename,
				}
				a.db.Create(&dbFile)
				message += "[" + dbFile.Name + "](" + a.config.URL + "/file/" + url.QueryEscape(dbFile.Name) + "?id=" + dbFile.UUID + ")"
			}
		}
	}
	dbMsg := Message{
		Content: message,
	}
	dbReport := Report{
		Messages: []Message{dbMsg},
		State:    ReportStateOpen,
	}
	err = a.db.Create(&dbReport).Error
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, "can't create report")
	}
	for _, m := range a.config.getMessengers() {
		err := m.SendMessage(fmt.Sprintf("Report #%d opened", dbReport.ID), message)
		if err != nil {
			log.Println("Can't send message:", err)
		}
	}
	c.Redirect(http.StatusFound, "/report?reporterToken="+dbReport.ReporterToken)
}

type ReportPageData struct {
	Config config
	Report Report

	IsAdministrator bool
}

func (a *App) reportRoute(c *gin.Context) {
	d := ReportPageData{
		Config: a.config,
		Report: Report{},
	}

	reporterToken := c.Query("reporterToken")
	administratorToken := c.Query("administratorToken")
	if administratorToken != "" {
		d.IsAdministrator = true
		if err := a.db.Where("administrator_token = ?", administratorToken).
			Preload("Messages").
			Find(&d.Report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else if reporterToken != "" {
		d.IsAdministrator = false
		if err := a.db.Where("reporter_token = ?", reporterToken).
			Preload("Messages").
			Find(&d.Report).Error; err != nil {
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
	res := File{}
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
	report := Report{}
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
	a.db.Create(&Message{
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
