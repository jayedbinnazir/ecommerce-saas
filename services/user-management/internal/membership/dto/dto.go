// Package dto holds the request/response payloads for the membership module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/membership/domain"
)

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Role   string `json:"role" binding:"required,oneof=MANAGER CUSTOMER"`
}

type ChangeRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=MANAGER CUSTOMER"`
}

type GrantPermissionRequest struct {
	PermissionID string `json:"permission_id" binding:"required,uuid"`
}

type MembershipResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	RoleID    uuid.UUID `json:"role_id"`
	RoleName  string    `json:"role_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func FromMembership(m *domain.Membership) MembershipResponse {
	return MembershipResponse{
		ID:        m.ID,
		TenantID:  m.TenantID,
		UserID:    m.UserID,
		RoleID:    m.RoleID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromView(v *domain.View) MembershipResponse {
	r := FromMembership(&v.Membership)
	r.RoleName = string(v.RoleName)
	return r
}

func FromViews(vs []domain.View) []MembershipResponse {
	out := make([]MembershipResponse, len(vs))
	for i := range vs {
		out[i] = FromView(&vs[i])
	}
	return out
}
