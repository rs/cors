// Package cors/wrapper/gin provides gin.HandlerFunc to handle CORS related
// requests as a wrapper of github.com/rs/cors handler.
package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

// Options is a configuration container to setup the CORS middleware.
type Options = cors.Options

// AllowAll creates a new CORS Gin middleware with permissive configuration
// allowing all origins with all standard methods with any header and
// credentials.
func AllowAll() gin.HandlerFunc {
	return Wrap(cors.AllowAll())
}

// Default creates a new CORS Gin middleware with default options.
func Default() gin.HandlerFunc {
	return Wrap(cors.Default())
}

// New creates a new CORS Gin middleware with the provided options.
func New(options Options) gin.HandlerFunc {
	return Wrap(cors.New(options))
}

// Wrap transforms given cors.Cors handler into Gin middleware.
func Wrap(c *cors.Cors) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		if !c.OptionPassthrough &&
			ctx.Request.Method == http.MethodOptions &&
			ctx.GetHeader("Access-Control-Request-Method") != "" {
			// Abort processing next Gin middlewares.
			ctx.AbortWithStatus(http.StatusOK)
		}
	}
}
