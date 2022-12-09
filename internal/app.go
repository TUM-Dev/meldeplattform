package internal

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
)

type App struct {
	engine *gin.Engine
	db     *gorm.DB

	config config

	template *template.Template
}

func NewApp() *App {
	return &App{
		engine: gin.New(),
	}
}

func (a *App) Run() error {
	a.engine.Use(gin.Logger(), gin.Recovery())
	_ = a.engine.SetTrustedProxies(nil)

	err := a.initCfg()
	if err != nil {
		return fmt.Errorf("init config: %v", err)
	}
	if a.config.Mode == "prod" {
		// strip clientip from requests:
		a.engine.Use(func(c *gin.Context) {
			c.Request.RemoteAddr = "<Client IP Redacted>"
		})
	}
	err = a.initDB()
	if err != nil {
		return fmt.Errorf("initDB: %w", err)
	}
	a.initRoutes()
	return a.engine.Run()
}
