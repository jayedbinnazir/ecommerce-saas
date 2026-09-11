// Package dto holds the request/response payloads for the mail module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/mail-service/internal/mail/domain"
)

// SendRequest is the internal body other services post to send one email.
type SendRequest struct {
	To       string         `json:"to" binding:"required,email"`
	Template string         `json:"template" binding:"required"`
	Data     map[string]any `json:"data"`
}

type MailResponse struct {
	ID        uuid.UUID `json:"id"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Template  string    `json:"template"`
	Status    string    `json:"status"`
	Error     *string   `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func FromMail(m *domain.Mail) MailResponse {
	return MailResponse{
		ID:        m.ID,
		To:        m.ToAddr,
		Subject:   m.Subject,
		Template:  m.Template,
		Status:    string(m.Status),
		Error:     m.Error,
		CreatedAt: m.CreatedAt,
	}
}

func FromMails(ms []domain.Mail) []MailResponse {
	out := make([]MailResponse, len(ms))
	for i := range ms {
		out[i] = FromMail(&ms[i])
	}
	return out
}
