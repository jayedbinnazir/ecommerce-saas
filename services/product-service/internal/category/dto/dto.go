// Package dto holds the request/response payloads for the category module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/category/domain"
)

type CreateCategoryRequest struct {
	Name     string  `json:"name" binding:"required,max=255"`
	Slug     string  `json:"slug" binding:"required,min=2,max=140"`
	ParentID *string `json:"parent_id" binding:"omitempty,uuid"`
	Position int     `json:"position"`
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=255"`
	Slug     *string `json:"slug" binding:"omitempty,min=2,max=140"`
	ParentID *string `json:"parent_id" binding:"omitempty,uuid"` // "" (empty string) clears the parent
	Position *int    `json:"position"`
}

func (r UpdateCategoryRequest) IsEmpty() bool {
	return r.Name == nil && r.Slug == nil && r.ParentID == nil && r.Position == nil
}

type CategoryResponse struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Position  int        `json:"position"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func FromCategory(c *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:        c.ID,
		TenantID:  c.TenantID,
		ParentID:  c.ParentID,
		Name:      c.Name,
		Slug:      c.Slug,
		Position:  c.Position,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func FromCategories(cs []domain.Category) []CategoryResponse {
	out := make([]CategoryResponse, len(cs))
	for i := range cs {
		out[i] = FromCategory(&cs[i])
	}
	return out
}
