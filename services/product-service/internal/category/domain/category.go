package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Category is a per-tenant catalog category. ParentID makes it a tree.
// Table "categories", UNIQUE (tenant_id, slug).
type Category struct {
	ID        uuid.UUID  `json:"id"         db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"  db:"tenant_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"` // NULLABLE, FK self ON DELETE SET NULL
	Name      string     `json:"name"       db:"name"`
	Slug      string     `json:"slug"       db:"slug"`
	Position  int        `json:"position"   db:"position"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, c *Category) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Category, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]Category, error)
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// Exists reports whether id belongs to tenantID (used when attaching products).
	Exists(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
	// CountExisting returns how many of ids belong to tenantID, in one query, so
	// callers can validate a whole set of category ids without an N+1 loop.
	CountExisting(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int, error)
}
