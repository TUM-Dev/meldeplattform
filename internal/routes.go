package internal

import (
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
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
	a.engine.StaticFS("/static", http.FS(static))
}

func (a *App) indexRoute(c *gin.Context) {
	err := a.template.ExecuteTemplate(c.Writer, "index.gohtml", a.config)
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) submitRoute(c *gin.Context) {

}
