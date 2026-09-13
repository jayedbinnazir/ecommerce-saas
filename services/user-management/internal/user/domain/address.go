package domain

import (
	"time"

	"github.com/google/uuid"
)

// Address maps to table "addresses".
// user_id REFERENCES users(id) ON DELETE CASCADE.
// created_at / updated_at are TIMESTAMPTZ, stored in UTC.
type Address struct {
	ID                uuid.UUID `json:"id"                       db:"id"`
	UserID            uuid.UUID `json:"user_id"                  db:"user_id"`
	Label             *string   `json:"label,omitempty"          db:"label"` // NULLABLE
	RecipientName     string    `json:"recipient_name"           db:"recipient_name"`
	Phone             string    `json:"phone"                    db:"phone"`
	AddressLine1      string    `json:"address_line_1"           db:"address_line_1"`
	AddressLine2      *string   `json:"address_line_2,omitempty" db:"address_line_2"` // NULLABLE
	City              string    `json:"city"                     db:"city"`
	State             *string   `json:"state,omitempty"          db:"state"`       // NULLABLE
	PostalCode        *string   `json:"postal_code,omitempty"    db:"postal_code"` // NULLABLE
	Country           string    `json:"country"                  db:"country"`     // ISO 3166-1 alpha-2
	IsDefaultShipping bool      `json:"is_default_shipping"      db:"is_default_shipping"`
	IsDefaultBilling  bool      `json:"is_default_billing"       db:"is_default_billing"`
	CreatedAt         time.Time `json:"created_at"               db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"               db:"updated_at"`
}
