package internal

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"io"
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
	a.template = template.Must(template.New("base").ParseFS(templates, "templates/*.gohtml"))
	a.engine.GET("/", a.indexRoute)
	a.engine.POST("/submit", a.submitRoute)
	a.engine.GET("/file/:name", a.getFile)
	a.engine.GET("/report", a.reportRoute)
	a.engine.StaticFS("/static", http.FS(static))
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", a.config)
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

	for i, field := range a.config.Content.Fields {
		message += "<b>" + field.Name + "</b><br>\n"

		// handle text-ish fields
		if field.Type != "file" {
			fieldResp := c.PostForm(fmt.Sprintf("%d", i))
			if fieldResp == "" && field.Required {
				c.AbortWithStatusJSON(http.StatusBadRequest, "required field not provided")
				return
			}
			message += fieldResp + "<br>\n<br>\n"
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
				message += "<a href=\"" + a.config.URL + "/file/" + url.QueryEscape(dbFile.Name) + "?id=" + dbFile.UUID + "\">" +
					dbFile.Name + "</a>" + "<br>\n"
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
		if err := a.db.Where("administrator_token = ?", administratorToken).Find(&d.Report).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	} else if reporterToken != "" {
		d.IsAdministrator = false
		if err := a.db.Debug().
			Where("reporter_token = ?", reporterToken).
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
