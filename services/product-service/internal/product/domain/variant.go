package domain

import (
	"time"

	"github.com/google/uuid"
)

// Variant is a purchasable form of a product (e.g. "Small / Red"). Price lives
// here. Stock does NOT — that belongs to the inventory service; SKU is the key
// other services join on.
// Table "product_variants", UNIQUE (tenant_id, sku).
type Variant struct {
	ID                  uuid.UUID `json:"id"                   db:"id"`
	ProductID           uuid.UUID `json:"product_id"           db:"product_id"`
	TenantID            uuid.UUID `json:"tenant_id"            db:"tenant_id"`
	SKU                 string    `json:"sku"                  db:"sku"`
	Title               *string   `json:"title,omitempty"      db:"title"`
	PriceCents          int64     `json:"price_cents"          db:"price_cents"`
	Currency            string    `json:"currency"             db:"currency"` // ISO 4217
	CompareAtPriceCents *int64    `json:"compare_at_price_cents,omitempty" db:"compare_at_price_cents"`
	WeightGrams         *int      `json:"weight_grams,omitempty" db:"weight_grams"`
	Barcode             *string   `json:"barcode,omitempty"    db:"barcode"`
	Position            int       `json:"position"             db:"position"`
	IsDefault           bool      `json:"is_default"           db:"is_default"`
	CreatedAt           time.Time `json:"created_at"           db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"           db:"updated_at"`
}
