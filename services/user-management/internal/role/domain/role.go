package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type RoleName string

const (
	SuperAdmin RoleName = "SUPER_ADMIN" // platform operator (not tenant-scoped)
	Admin      RoleName = "ADMIN"       // tenant owner / administrator
	Manager    RoleName = "MANAGER"     // staff assigned by a tenant admin
	Customer   RoleName = "CUSTOMER"    // shopper
)

func (n RoleName) Valid() bool {
	switch n {
	case SuperAdmin, Admin, Manager, Customer:
		return true
	default:
		return false
	}
}

// TenantScoped reports whether the role is assignable within a tenant membership.
func (n RoleName) TenantScoped() bool {
	return n == Admin || n == Manager || n == Customer
}

// Role is a named set of capabilities. Table "roles".
type Role struct {
	ID          uuid.UUID `json:"id"                    db:"id"`
	Name        RoleName  `json:"name"                 db:"name"`         // UNIQUE
	Description *string   `json:"description,omitempty" db:"description"` // NULLABLE
	CreatedAt   time.Time `json:"created_at"            db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"            db:"updated_at"`
}

type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, name RoleName) (*Role, error)
	List(ctx context.Context, page platform.Page) ([]Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}
