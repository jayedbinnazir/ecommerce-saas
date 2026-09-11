// Package dto holds the request/response payloads for the notification module.
package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/notification-service/internal/notification/domain"
)

// CreateRequest is the internal body other services post. If Email and
// EmailTemplate are both set, mail-service is also asked to send that template.
type CreateRequest struct {
	UserID        uuid.UUID      `json:"user_id" binding:"required"`
	TenantID      *string        `json:"tenant_id" binding:"omitempty,uuid"`
	Type          string         `json:"type" binding:"required,max=64"`
	Title         string         `json:"title" binding:"required,max=200"`
	Body          string         `json:"body" binding:"required,max=2000"`
	Data          map[string]any `json:"data"`
	Email         string         `json:"email" binding:"omitempty,email"`
	EmailTemplate string         `json:"email_template" binding:"omitempty,max=64"`
}

// MarkReadRequest marks specific ids read, or every unread one when All is true.
type MarkReadRequest struct {
	IDs []string `json:"ids" binding:"omitempty,dive,uuid"`
	All bool     `json:"all"`
}

type NotificationResponse struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	TenantID  *uuid.UUID      `json:"tenant_id,omitempty"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data,omitempty"`
	Read      bool            `json:"read"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

func FromNotification(n *domain.Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		TenantID:  n.TenantID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		Read:      n.ReadAt != nil,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}

func FromNotifications(ns []domain.Notification) []NotificationResponse {
	out := make([]NotificationResponse, len(ns))
	for i := range ns {
		out[i] = FromNotification(&ns[i])
	}
	return out
}
