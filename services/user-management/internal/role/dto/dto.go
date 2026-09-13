// Package dto holds the request/response payloads for the role module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
)

type CreateRoleRequest struct {
	Name        string  `json:"name" binding:"required,oneof=SUPER_ADMIN ADMIN MANAGER CUSTOMER"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name" binding:"omitempty,oneof=SUPER_ADMIN ADMIN MANAGER CUSTOMER"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

func (r UpdateRoleRequest) IsEmpty() bool { return r.Name == nil && r.Description == nil }

type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func FromRole(r *domain.Role) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        string(r.Name),
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func FromRoles(rs []domain.Role) []RoleResponse {
	out := make([]RoleResponse, len(rs))
	for i := range rs {
		out[i] = FromRole(&rs[i])
	}
	return out
}
