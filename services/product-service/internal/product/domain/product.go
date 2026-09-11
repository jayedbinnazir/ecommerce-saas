package domain

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft    Status = "DRAFT"
	StatusActive   Status = "ACTIVE"
	StatusArchived Status = "ARCHIVED"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusArchived:
		return true
	default:
		return false
	}
}

// Product is a catalog item owned by a tenant (another service's aggregate, so
// tenant_id is a plain reference, not a foreign key).
// Table "products", UNIQUE (tenant_id, slug).
type Product struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	TenantID    uuid.UUID `json:"tenant_id"   db:"tenant_id"`
	Name        string    `json:"name"        db:"name"`
	Slug        string    `json:"slug"        db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	Status      Status    `json:"status"      db:"status"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
}

// Detail is a product with everything a storefront needs to render it.
type Detail struct {
	Product
	Variants   []Variant   `json:"variants"`
	Images     []Image     `json:"images"`
	CategoryID []uuid.UUID `json:"category_ids"`

	// Options are the customer-selectable axes (from the product's categories),
	// each with its choices. Specs are the display-only values for this product.
	Options []OptionAxis `json:"-"`
	Specs   []SpecEntry  `json:"-"`
	// VariantOptions maps a variant id to {attribute code: chosen value}.
	VariantOptions map[uuid.UUID]map[string]string `json:"-"`
}
