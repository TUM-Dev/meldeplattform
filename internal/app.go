package internal

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
	"net/http"
)

type App struct {
	engine *gin.Engine
	db     *gorm.DB

	config config

	template *template.Template
}

func NewApp() *App {
	return &App{
		engine: gin.Default(),
	}
}

func (a *App) Run() error {
	err := a.initCfg()
	if err != nil {
		return fmt.Errorf("init config: %v", err)
	}
	err = a.initDB()
	if err != nil {
		return fmt.Errorf("initDB: %w", err)
	}
	a.initRoutes()
	return a.engine.Run()
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
