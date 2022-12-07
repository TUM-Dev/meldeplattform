package internal

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
)

type App struct {
	engine *gin.Engine
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
	a.initRoutes()
	return a.engine.Run()
}
