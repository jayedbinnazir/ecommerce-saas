// Package domain holds the attribute aggregate: per-category field definitions
// that are either a customer-selectable VARIANT axis or a display-only SPEC.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Role decides what an attribute is for.
type Role string

const (
	// RoleVariant: the customer picks it; each combination is a priced variant.
	RoleVariant Role = "VARIANT"
	// RoleSpec: display only, shown on the product page.
	RoleSpec Role = "SPEC"
)

func (r Role) Valid() bool { return r == RoleVariant || r == RoleSpec }

// Attribute is a field definition attached to a category. Table
// "product_attributes", UNIQUE (category_id, code).
type Attribute struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	TenantID   uuid.UUID `json:"tenant_id"   db:"tenant_id"`
	CategoryID uuid.UUID `json:"category_id" db:"category_id"`
	Name       string    `json:"name"        db:"name"` // "RAM"
	Code       string    `json:"code"        db:"code"` // "ram"
	Role       Role      `json:"role"        db:"role"`
	Position   int       `json:"position"    db:"position"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"  db:"updated_at"`
}

// Value is one allowed choice for an attribute ("12GB"). Table
// "product_attribute_values", UNIQUE (attribute_id, value).
type Value struct {
	ID          uuid.UUID `json:"id"           db:"id"`
	AttributeID uuid.UUID `json:"attribute_id" db:"attribute_id"`
	Value       string    `json:"value"        db:"value"`
	Position    int       `json:"position"     db:"position"`
	CreatedAt   time.Time `json:"created_at"   db:"created_at"`
}

// Detail is an attribute together with its values (management + catalog view).
type Detail struct {
	Attribute
	Values []Value `json:"values"`
}

type Repository interface {
	CreateAttribute(ctx context.Context, a *Attribute) error
	GetAttribute(ctx context.Context, tenantID, id uuid.UUID) (*Attribute, error)
	ListByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) ([]Attribute, error)
	UpdateAttribute(ctx context.Context, a *Attribute) error
	DeleteAttribute(ctx context.Context, tenantID, id uuid.UUID) error

	AddValue(ctx context.Context, v *Value) error
	DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error

	// ValuesByAttributes returns every value for the given attributes in one
	// query, grouped by attribute id (no N+1 when rendering a product).
	ValuesByAttributes(ctx context.Context, attributeIDs []uuid.UUID) (map[uuid.UUID][]Value, error)

	// ListByCategories returns the distinct attributes of several categories in
	// one query — used to resolve a product's attribute set from its categories.
	ListByCategories(ctx context.Context, categoryIDs []uuid.UUID) ([]Attribute, error)
}
