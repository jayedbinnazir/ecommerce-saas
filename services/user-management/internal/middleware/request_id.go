package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the header carrying the per-request correlation id.
const RequestIDHeader = "X-Request-ID"

// RequestID reuses an incoming X-Request-ID or generates one, and echoes it back
// on the response so clients and logs can correlate a request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}
