package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
)

// InternalKeyHeader carries the shared service-to-service secret.
const InternalKeyHeader = "X-Internal-Key"

// RequireInternalKey allows a request only if it presents the configured internal
// key. Used for the /api/v1/internal routes other services call. Empty key
// closes those routes.
func RequireInternalKey(key string) gin.HandlerFunc {
	want := []byte(key)
	return func(c *gin.Context) {
		if len(want) == 0 {
			_ = c.Error(httpx.Forbidden("internal API is not enabled"))
			c.Abort()
			return
		}
		if subtle.ConstantTimeCompare([]byte(c.GetHeader(InternalKeyHeader)), want) != 1 {
			_ = c.Error(httpx.Unauthorized("invalid internal API key"))
			c.Abort()
			return
		}
		c.Next()
	}
}
