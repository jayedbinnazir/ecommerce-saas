package httpx

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/notification-service/internal/platform"
)

// Principal returns the authenticated caller. If none is present it pushes a 401
// onto the gin context and returns ok=false; callers should return immediately.
func Principal(c *gin.Context) (platform.Principal, bool) {
	p, ok := platform.PrincipalFromContext(c.Request.Context())
	if !ok {
		_ = c.Error(Unauthorized("authentication required"))
		return platform.Principal{}, false
	}
	return p, true
}
