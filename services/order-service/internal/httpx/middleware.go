package httpx

import "github.com/gin-gonic/gin"

// ErrorHandler renders the last error a handler reported with c.Error(err) as
// the standard error response (see RenderError).
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		RenderError(c, c.Errors.Last().Err)
	}
}
