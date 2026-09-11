package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/mail-service/internal/httpx"
)

// InternalKeyHeader carries the shared service-to-service secret.
const InternalKeyHeader = "X-Internal-Key"

// RequireInternalKey allows a request only if it presents the configured internal
// key. If no key is configured the route is closed.
func RequireInternalKey(key string) gin.HandlerFunc {
	want := []byte(key)
	return func(c *gin.Context) {
		if len(want) == 0 {
			_ = c.Error(httpx.Forbidden("internal API is not enabled"))
			c.Abort()
			return
		}
		got := []byte(c.GetHeader(InternalKeyHeader))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			_ = c.Error(httpx.Unauthorized("invalid internal API key"))
			c.Abort()
			return
		}
		c.Next()
	}
}
