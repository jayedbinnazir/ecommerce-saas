// Package dto holds the request/response payloads for the permission module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/permission/domain"
)

type CreatePermissionRequest struct {
	Name        string  `json:"name" binding:"required,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

type UpdatePermissionRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

func (r UpdatePermissionRequest) IsEmpty() bool { return r.Name == nil && r.Description == nil }

// GrantRequest assigns an existing permission (by id) to a role or membership.
type GrantRequest struct {
	PermissionID string `json:"permission_id" binding:"required,uuid"`
}

type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func FromPermission(p *domain.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func FromPermissions(ps []domain.Permission) []PermissionResponse {
	out := make([]PermissionResponse, len(ps))
	for i := range ps {
		out[i] = FromPermission(&ps[i])
	}
	return out
}
