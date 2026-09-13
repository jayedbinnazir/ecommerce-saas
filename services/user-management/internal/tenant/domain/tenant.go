package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusSuspended Status = "SUSPENDED"
	StatusInactive  Status = "INACTIVE"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusSuspended, StatusInactive:
		return true
	default:
		return false
	}
}

// Tenant is a single ecommerce store. OwnerUserID is the user who created it and
// holds the ADMIN membership. Table "tenants".
type Tenant struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	Name        string    `json:"name"        db:"name"`
	Slug        string    `json:"slug"        db:"slug"` // UNIQUE, URL-safe
	Status      Status    `json:"status"      db:"status"`
	OwnerUserID uuid.UUID `json:"owner_user_id" db:"owner_user_id"` // REFERENCES users(id) ON DELETE RESTRICT
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Tenant, error)
	List(ctx context.Context, page platform.Page) ([]Tenant, error)
	Update(ctx context.Context, t *Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SubscriptionGuard is implemented by the billing module: it fails when the user
// has no active subscription entitling them to create a store.
type SubscriptionGuard interface {
	RequireActiveSubscription(ctx context.Context, userID uuid.UUID) error
}

// OwnerEnroller is implemented by the membership module: it creates the owner's
// ADMIN membership for a freshly created tenant. It receives the tenant service's
// transaction so store creation and enrollment commit together.
type OwnerEnroller interface {
	EnrollOwnerAsAdmin(ctx context.Context, exec platform.DBTX, tenantID, userID uuid.UUID) error
}
