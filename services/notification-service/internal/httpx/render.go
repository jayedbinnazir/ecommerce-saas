package httpx

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/notification-service/internal/platform"
)

type errorResponse struct {
	Status string      `json:"status"` // always "error"
	Error  errorDetail `json:"error"`
}

type errorDetail struct {
	Service   string       `json:"service"`
	Code      Code         `json:"code"`
	Message   string       `json:"message"`
	Method    string       `json:"method"`
	Path      string       `json:"path"`
	RequestID string       `json:"request_id,omitempty"`
	Timestamp string       `json:"timestamp"` // RFC3339, UTC
	Fields    []FieldError `json:"fields,omitempty"`
}

// RequestIDFromContext returns the id set by the RequestID middleware, if any.
func RequestIDFromContext(c *gin.Context) string {
	return c.GetString("request_id")
}

// RenderError classifies err and writes the standard error response. Shared by
// the ErrorHandler, Recovery and RateLimit middleware.
func RenderError(c *gin.Context, err error) {
	appErr := Classify(err)
	requestID := RequestIDFromContext(c)

	if appErr.Status >= 500 {
		log.Printf("[error] service=%s request_id=%s %s %s -> %d %s: %v",
			platform.ServiceName, requestID,
			c.Request.Method, c.Request.URL.Path,
			appErr.Status, appErr.Code, appErr.Error())
	}

	c.AbortWithStatusJSON(appErr.Status, errorResponse{
		Status: "error",
		Error: errorDetail{
			Service:   platform.ServiceName,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			RequestID: requestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Fields:    appErr.Fields,
		},
	})
}
