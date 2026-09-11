// Package routes wires the notification module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/notification-service/internal/httpx"
	"github.com/jayedbinnazir/notification-service/internal/mailclient"
	"github.com/jayedbinnazir/notification-service/internal/middleware"
	httphandler "github.com/jayedbinnazir/notification-service/internal/notification/handler/http"
	"github.com/jayedbinnazir/notification-service/internal/notification/repository"
	"github.com/jayedbinnazir/notification-service/internal/notification/services"
)

// Register mounts:
//   - POST /internal/notifications                service-to-service (X-Internal-Key)
//   - GET  /notifications          (?unread=true) the caller's own
//   - GET  /notifications/unread-count
//   - POST /notifications/read     { ids:[...] } or { all:true }
func Register(rg *gin.RouterGroup, db *sql.DB, mail *mailclient.Client, guards httpx.Guards, internalKey string) {
	h := httphandler.New(services.New(repository.New(db), mail))

	rg.POST("/internal/notifications", middleware.RequireInternalKey(internalKey), h.Create)

	me := rg.Group("/notifications", guards.Authenticated)
	{
		me.GET("", h.List)
		me.GET("/unread-count", h.UnreadCount)
		me.POST("/read", h.MarkRead)
	}
}
