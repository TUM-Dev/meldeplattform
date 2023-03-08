package internal

import (
	_ "embed"
	"fmt"
	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"github.com/TUM-Dev/meldeplattform/pkg/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
	"net/http"
)

type App struct {
	engine *gin.Engine
	db     *gorm.DB
	i18n   i18n.I18n

	config model.Config

	template *template.Template
}

func NewApp() *App {
	a := &App{
		engine: gin.New(),
	}

	i, err := i18n.New(i18nStr)
	if err != nil {
		panic(err)
	}
	a.i18n = i
	a.engine.Use(gin.Logger(), gin.Recovery())
	_ = a.engine.SetTrustedProxies(nil)

	err = a.initCfg()
	if err != nil {
		panic(fmt.Errorf("init config: %v", err))
	}
	if a.config.Mode == "prod" {
		// strip clientip from requests:
		a.engine.Use(func(c *gin.Context) {
			c.Request.RemoteAddr = "<Client IP Redacted>"
		})
	}
	err = a.initDB()
	if err != nil {
		panic(fmt.Errorf("initDB: %w", err))
	}
	a.initRoutes()
	return a
}

func (a *App) Run() error {
	return a.engine.Run()
}

func (a *App) setLang(c *gin.Context) {
	reqLang := c.Request.URL.Query().Get("lang")
	c.SetCookie("lang", reqLang, 60*60*24*365, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

//go:embed web/i18n.json
var i18nStr string
