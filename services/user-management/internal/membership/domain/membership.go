package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
)

// Membership links one user to one tenant with one role. A user may hold many
// memberships across tenants. Join table "memberships", UNIQUE (tenant_id, user_id).
type Membership struct {
	ID        uuid.UUID `json:"id"         db:"id"`
	TenantID  uuid.UUID `json:"tenant_id"  db:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"    db:"user_id"`
	RoleID    uuid.UUID `json:"role_id"    db:"role_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// View is a Membership joined with its role name, for authorization decisions.
type View struct {
	Membership
	RoleName roledomain.RoleName `json:"role_name"`
}

type Repository interface {
	Create(ctx context.Context, m *Membership) error
	GetByID(ctx context.Context, id uuid.UUID) (*Membership, error)
	GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*Membership, error)
	ResolveView(ctx context.Context, tenantID, userID uuid.UUID) (*View, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]View, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]View, error)
	// UpdateRole and Delete are scoped by tenantID too — every call site
	// already resolves the membership via GetByTenantAndUser first, but the
	// query defends itself too rather than trusting a bare membership id.
	UpdateRole(ctx context.Context, tenantID, id, roleID uuid.UUID) (*Membership, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
