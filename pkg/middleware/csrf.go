package middleware

import (
	"net/http"
	"os"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	adapter "github.com/gwatts/gin-adapter"
)

var csrfMd func(http.Handler) http.Handler

var csrfSecret = uniuri.NewLen(32)

func init() {
	isDev := os.Getenv("GO_ENV") == "DEV"
	useSecure := !isDev

	csrfMd = csrf.Protect([]byte(csrfSecret),
		csrf.MaxAge(0),
		csrf.Secure(useSecure),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "Forbidden - CSRF token invalid"}`))
		})),
	)
}

func CSRF() gin.HandlerFunc {
	return adapter.Wrap(csrfMd)
}
