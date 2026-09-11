// Package dto holds the request/response payloads for the attribute module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/attributes/domain"
)

// ---- requests ----

type CreateAttributeRequest struct {
	Name     string `json:"name" binding:"required,max=120"`
	Code     string `json:"code" binding:"required,min=2,max=60"`
	Role     string `json:"role" binding:"required,oneof=VARIANT SPEC"`
	Position int    `json:"position"`
}

type UpdateAttributeRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=120"`
	Code     *string `json:"code" binding:"omitempty,min=2,max=60"`
	Role     *string `json:"role" binding:"omitempty,oneof=VARIANT SPEC"`
	Position *int    `json:"position"`
}

func (r UpdateAttributeRequest) IsEmpty() bool {
	return r.Name == nil && r.Code == nil && r.Role == nil && r.Position == nil
}

type AddValueRequest struct {
	Value    string `json:"value" binding:"required,max=120"`
	Position int    `json:"position"`
}

// ---- responses ----

type ValueResponse struct {
	ID       uuid.UUID `json:"id"`
	Value    string    `json:"value"`
	Position int       `json:"position"`
}

func FromValue(v *domain.Value) ValueResponse {
	return ValueResponse{ID: v.ID, Value: v.Value, Position: v.Position}
}

func FromValues(vs []domain.Value) []ValueResponse {
	out := make([]ValueResponse, len(vs))
	for i := range vs {
		out[i] = FromValue(&vs[i])
	}
	return out
}

type AttributeResponse struct {
	ID         uuid.UUID       `json:"id"`
	CategoryID uuid.UUID       `json:"category_id"`
	Name       string          `json:"name"`
	Code       string          `json:"code"`
	Role       string          `json:"role"`
	Position   int             `json:"position"`
	Values     []ValueResponse `json:"values"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func FromDetail(d *domain.Detail) AttributeResponse {
	return AttributeResponse{
		ID:         d.ID,
		CategoryID: d.CategoryID,
		Name:       d.Name,
		Code:       d.Code,
		Role:       string(d.Role),
		Position:   d.Position,
		Values:     FromValues(d.Values),
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func FromDetails(ds []domain.Detail) []AttributeResponse {
	out := make([]AttributeResponse, len(ds))
	for i := range ds {
		out[i] = FromDetail(&ds[i])
	}
	return out
}
