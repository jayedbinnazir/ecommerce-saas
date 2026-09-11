package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"status": "success", "data": data})
}

func OK(c *gin.Context, data any)      { success(c, http.StatusOK, data) }
func Created(c *gin.Context, data any) { success(c, http.StatusCreated, data) }
func NoContent(c *gin.Context)         { c.Status(http.StatusNoContent) }

// List renders a paginated collection envelope.
func List(c *gin.Context, items any, limit, offset int) {
	success(c, http.StatusOK, gin.H{"items": items, "limit": limit, "offset": offset})
}
