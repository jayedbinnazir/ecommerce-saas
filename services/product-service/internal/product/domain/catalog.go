package domain

import "github.com/google/uuid"

// This file holds the "resolved catalog" shapes: a product's attribute set comes
// from its categories (see the attributes module), but the storefront wants it
// stitched onto the product. These types are what the service assembles for the
// product-detail response.

// OptionValue is one selectable choice on a variant axis ("12GB").
type OptionValue struct {
	ID    uuid.UUID
	Value string
}

// OptionAxis is a customer-selectable attribute plus its choices. Picking one
// value per axis identifies a single variant (and therefore a price).
type OptionAxis struct {
	AttributeID uuid.UUID
	Code        string // "ram"
	Name        string // "RAM"
	Values      []OptionValue
}

// SpecEntry is one display-only specification value for a product.
type SpecEntry struct {
	Code  string // "chipset"
	Name  string // "Chipset"
	Value string // "Snapdragon 8 Gen 3"
}

// ProductSpec is a stored display-spec value (write side).
type ProductSpec struct {
	ProductID   uuid.UUID
	AttributeID uuid.UUID
	Value       string
}

// VariantOption ties one variant to its chosen value for one axis (write side).
type VariantOption struct {
	VariantID        uuid.UUID
	AttributeID      uuid.UUID
	AttributeValueID uuid.UUID
}
