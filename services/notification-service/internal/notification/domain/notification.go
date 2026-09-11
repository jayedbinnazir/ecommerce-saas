package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/notification-service/internal/platform"
)

// Notification is one in-app message for one user. Table "notifications".
type Notification struct {
	ID        uuid.UUID       `json:"id"         db:"id"`
	UserID    uuid.UUID       `json:"user_id"    db:"user_id"`
	TenantID  *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	Type      string          `json:"type"       db:"type"` // e.g. "order.confirmed", "order.shipped"
	Title     string          `json:"title"      db:"title"`
	Body      string          `json:"body"       db:"body"`
	Data      json.RawMessage `json:"data,omitempty" db:"data"` // arbitrary JSON payload (order id, etc.)
	ReadAt    *time.Time      `json:"read_at,omitempty" db:"read_at"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// ListFilter narrows a user's notification list.
type ListFilter struct {
	UserID     uuid.UUID
	UnreadOnly bool
	Page       platform.Page
}

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	ListByUser(ctx context.Context, f ListFilter) ([]Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	// MarkRead marks the given ids read for one user, in a single query.
	MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (int, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error)
}
