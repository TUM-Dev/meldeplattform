package middleware

import (
	"net/http"

	"github.com/dchest/uniuri"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	adapter "github.com/gwatts/gin-adapter"
)

var csrfMd func(http.Handler) http.Handler

func init() {
	csrfMd = csrf.Protect([]byte(uniuri.NewLen(32)),
		csrf.MaxAge(0),
		csrf.Secure(false),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "Forbidden - CSRF token invalid"}`))
		})),
	)
}

func CSRF() gin.HandlerFunc {
	return adapter.Wrap(csrfMd)
}
