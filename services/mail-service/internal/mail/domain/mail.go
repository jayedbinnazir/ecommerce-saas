package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Status is the delivery state of one email.
type Status string

const (
	StatusSent   Status = "SENT"
	StatusFailed Status = "FAILED"
)

// Mail is one attempted email, kept as an audit log. Table "mails".
type Mail struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	ToAddr    string    `json:"to"         db:"to_addr"`
	Subject   string    `json:"subject"    db:"subject"`
	Template  string    `json:"template"   db:"template"`
	Status    Status    `json:"status"     db:"status"`
	Error     *string   `json:"error,omitempty" db:"error"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, m *Mail) error
	ListByRecipient(ctx context.Context, toAddr string, limit int) ([]Mail, error)
}
