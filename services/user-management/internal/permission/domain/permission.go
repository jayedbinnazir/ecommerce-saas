package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

// Permission is an atomic capability string, e.g. "product:write", "order:read".
// Table "permissions".
type Permission struct {
	ID          uuid.UUID `json:"id"                    db:"id"`
	Name        string    `json:"name"                 db:"name"`         // UNIQUE
	Description *string   `json:"description,omitempty" db:"description"` // NULLABLE
	CreatedAt   time.Time `json:"created_at"            db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"            db:"updated_at"`
}

// RolePermission is a baseline grant: every member with the role gets it.
// Join table "role_permissions", PRIMARY KEY (role_id, permission_id),
// both FKs ON DELETE CASCADE.
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id"       db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

// MembershipPermission is an extra grant to one specific tenant member, layered
// on top of whatever their role already provides.
// Join table "membership_permissions", PRIMARY KEY (membership_id, permission_id),
// both FKs ON DELETE CASCADE.
type MembershipPermission struct {
	MembershipID uuid.UUID `json:"membership_id" db:"membership_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
}

type PermissionRepository interface {
	Create(ctx context.Context, p *Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	GetByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context, page platform.Page) ([]Permission, error)
	Update(ctx context.Context, p *Permission) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// GrantRepository manages both grant layers and answers effective-permission
// queries used for authorization.
type GrantRepository interface {
	AssignToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RevokeFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	ListByRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error)

	AssignToMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error
	RevokeFromMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error
	ListByMembership(ctx context.Context, membershipID uuid.UUID) ([]Permission, error)

	// EffectiveForMembership returns role grants ∪ membership grants as a set of
	// permission names.
	EffectiveForMembership(ctx context.Context, membershipID uuid.UUID) ([]string, error)
}
