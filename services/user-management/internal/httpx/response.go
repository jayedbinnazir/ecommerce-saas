package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// success writes the canonical success envelope:
//
//	{ "status": "success", "data": ... }
func success(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{
		"status": "success",
		"data":   data,
	})
}

// OK renders 200 with a data payload.
func OK(c *gin.Context, data any) { success(c, http.StatusOK, data) }

// Created renders 201 with a data payload.
func Created(c *gin.Context, data any) { success(c, http.StatusCreated, data) }

// NoContent renders 204 with an empty body.
func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

// List renders 200 with a paginated collection envelope.
func List(c *gin.Context, items any, limit, offset int) {
	success(c, http.StatusOK, gin.H{
		"items":  items,
		"limit":  limit,
		"offset": offset,
	})
}
