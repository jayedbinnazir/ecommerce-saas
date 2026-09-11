package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/platform"
)

// Recovery catches any panic, logs the stack, and returns the standard error
// response instead of dropping the connection.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[panic] service=%s request_id=%s %s %s: %v\n%s",
					platform.ServiceName, httpx.RequestIDFromContext(c),
					c.Request.Method, c.Request.URL.Path, r, debug.Stack())
				httpx.RenderError(c, httpx.Internal("internal server error"))
			}
		}()
		c.Next()
	}
}
