package httpx

import "github.com/gin-gonic/gin"

// ErrorHandler is the centralized error-rendering middleware. Every module's
// handlers report failures with c.Error(err) and return; this middleware turns
// the last error into the standard error response (see RenderError).
//
// Response shape:
//
//	{
//	  "status": "error",
//	  "error": {
//	    "service":    "user-management",
//	    "code":       "VALIDATION_ERROR",
//	    "message":    "one or more fields are invalid",
//	    "method":     "POST",
//	    "path":       "/api/v1/auth/register",
//	    "request_id": "d9db17c2-...",
//	    "timestamp":  "2026-09-08T10:08:32Z",
//	    "fields":     [ { "field": "email", "message": "must be a valid email address" } ]
//	  }
//	}
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		RenderError(c, c.Errors.Last().Err)
	}
}
