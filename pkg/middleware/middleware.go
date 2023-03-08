package middleware

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
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

func InitTemplateBase(tr i18n.I18n, config model.Config, db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		lang := c.GetString("lang")
		base := model.Base{
			Lang:     lang,
			Tr:       tr,
			Config:   config,
			LoggedIn: false,
		}
		db.Preload(clause.Associations).Find(&base.Topics)
		cookie, err := c.Cookie("jwt")
		if err == nil {
			token, err := jwt.ParseWithClaims(cookie, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
				return keys.Leaf.PublicKey, nil
			})
			if err != nil {
				fmt.Println("JWT parsing error: ", err)
				c.SetCookie("jwt", "", -1, "/", "", false, true)
				return
			} else {
				base.LoggedIn = true
				base.Name = token.Claims.(*TokenClaims).Name
				base.Email = token.Claims.(*TokenClaims).Mail
				base.UID = token.Claims.(*TokenClaims).UID
				for _, user := range config.Content.AdminUsers {
					if user == base.UID {
						base.IsAdmin = true
						break
					}
				}
			}
		}
		c.Set("base", base)
	}
}

func AuthAdmin() func(c *gin.Context) {
	return func(c *gin.Context) {
		base := c.MustGet("base").(model.Base)
		if !base.IsAdmin {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

func AuthAdminOfTopic(db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		base := c.MustGet("base").(model.Base)
		if !base.IsAdmin {
			topicID := c.Param("topicID")
			var topic model.Topic
			db.Preload(clause.Associations).First(&topic, topicID)
			isAdmin := false
			for _, admin := range topic.Admins {
				if admin.UserID == base.UID {
					isAdmin = true
					break
				}
			}
			if !isAdmin {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
		}
	}
}
