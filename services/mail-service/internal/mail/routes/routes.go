// Package routes wires the mail module. Everything is internal (X-Internal-Key);
// mail-service has no public or user-facing API.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	httphandler "github.com/jayedbinnazir/mail-service/internal/mail/handler/http"
	"github.com/jayedbinnazir/mail-service/internal/mail/repository"
	"github.com/jayedbinnazir/mail-service/internal/mail/services"
	"github.com/jayedbinnazir/mail-service/internal/middleware"
	"github.com/jayedbinnazir/mail-service/internal/smtp"
)

func Register(rg *gin.RouterGroup, db *sql.DB, sender smtp.Sender, internalKey string) {
	h := httphandler.New(services.New(repository.New(db), sender))

	internal := rg.Group("/internal/mail", middleware.RequireInternalKey(internalKey))
	{
		internal.POST("", h.Send)
		internal.GET("", h.List)
		internal.GET("/templates", h.Templates)
	}
}
