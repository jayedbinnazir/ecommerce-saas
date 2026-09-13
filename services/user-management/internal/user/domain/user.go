package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a single platform identity. A user may own or belong to many tenants
// (see the membership module); IsSuperAdmin marks a platform-level operator.
//
// Table "users". created_at / updated_at are TIMESTAMPTZ, stored in UTC.
type User struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	Name         string    `json:"name"          db:"name"`
	Email        string    `json:"email"         db:"email"`         // UNIQUE, stored lower-cased
	Phone        *string   `json:"phone,omitempty" db:"phone"`       // NULLABLE
	PasswordHash *string   `json:"-"             db:"password_hash"` // NULLABLE: OAuth-only accounts
	IsSuperAdmin bool      `json:"is_super_admin" db:"is_super_admin"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}
