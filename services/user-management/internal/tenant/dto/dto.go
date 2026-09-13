// Package dto holds the request/response payloads for the tenant module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
)

// CreateTenantRequest is the "register your store" payload.
type CreateTenantRequest struct {
	Name string `json:"name" binding:"required,max=255"`
	Slug string `json:"slug" binding:"required,min=3,max=100"`
}

type UpdateTenantRequest struct {
	Name   *string `json:"name" binding:"omitempty,max=255"`
	Status *string `json:"status" binding:"omitempty,oneof=ACTIVE SUSPENDED INACTIVE"`
}

func (r UpdateTenantRequest) IsEmpty() bool { return r.Name == nil && r.Status == nil }

type TenantResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Status      string    `json:"status"`
	OwnerUserID uuid.UUID `json:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func FromTenant(t *domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:          t.ID,
		Name:        t.Name,
		Slug:        t.Slug,
		Status:      string(t.Status),
		OwnerUserID: t.OwnerUserID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func FromTenants(ts []domain.Tenant) []TenantResponse {
	out := make([]TenantResponse, len(ts))
	for i := range ts {
		out[i] = FromTenant(&ts[i])
	}
	return out
}
