package middleware

import (
	"strings"

	"github.com/TUM-Dev/meldeplattform/pkg/i18n"
	"github.com/TUM-Dev/meldeplattform/pkg/model"

	"github.com/gin-gonic/gin"
)

func InitI18n(c *gin.Context) {
	lang := "en" // default to english
	cookie, err := c.Request.Cookie("lang")
	if err == nil {
		switch cookie.Value {
		case "en":
			lang = "en"
		case "de":
			lang = "de"
		default:
			lang = "en"
			// unset if illegal value
			c.SetCookie("lang", "", -1, "/", "", false, true)
		}
	} else {
		// get preferred language from header
		h := c.Request.Header.Get("Accept-Language")
		if strings.HasPrefix(h, "de") {
			lang = "de"
		} else {
			lang = "en"
		}
	}

	c.Set("lang", lang)
	switch lang {
	case "de":
		c.Header("Content-Language", "de-DE")
	case "em":
		c.Header("Content-Language", "en-US")
	}
}

func InitTemplateBase(tr i18n.I18n, config model.Config) func(c *gin.Context) {
	return func(c *gin.Context) {
		lang := c.GetString("lang")
		base := model.Base{
			Lang:   lang,
			Tr:     tr,
			Config: config,
		}
		c.Set("base", base)
	}
}
