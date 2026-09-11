// Package httphandler contains the Gin HTTP handlers for the mail module.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/mail-service/internal/httpx"
	"github.com/jayedbinnazir/mail-service/internal/mail/dto"
	"github.com/jayedbinnazir/mail-service/internal/mail/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// Send — POST /internal/mail
func (h *Handler) Send(c *gin.Context) {
	var req dto.SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	mail, err := h.svc.Send(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromMail(mail))
}

// List — GET /internal/mail?to=<addr>&limit=<n>
func (h *Handler) List(c *gin.Context) {
	to := c.Query("to")
	if to == "" {
		_ = c.Error(httpx.BadRequest("query parameter to is required"))
		return
	}
	limit, _ := httpx.Pagination(c)
	mails, err := h.svc.ListByRecipient(c.Request.Context(), to, limit)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromMails(mails))
}

// Templates — GET /internal/mail/templates
func (h *Handler) Templates(c *gin.Context) {
	httpx.OK(c, gin.H{"templates": h.svc.Templates()})
}
