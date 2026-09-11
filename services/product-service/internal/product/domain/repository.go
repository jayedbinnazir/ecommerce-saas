package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
)

// ListFilter narrows a product listing.
type ListFilter struct {
	TenantID   uuid.UUID
	Status     *Status    // nil = any
	CategoryID *uuid.UUID // nil = any
	Search     string     // matches name/slug, empty = any
	Page       platform.Page
}

type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Product, error)
	GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*Product, error)
	List(ctx context.Context, f ListFilter) ([]Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type VariantRepository interface {
	Create(ctx context.Context, v *Variant) error
	// GetByID, Update, Delete and ClearDefault are all scoped by tenantID too
	// (not just id/productID) — the row is always reached from a caller that
	// already knows the tenant, but the query defends itself too.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Variant, error)
	ListByProduct(ctx context.Context, productID uuid.UUID) ([]Variant, error)
	CountByProduct(ctx context.Context, productID uuid.UUID) (int, error)
	Update(ctx context.Context, v *Variant) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	// ClearDefault unsets the current default variant for a product.
	ClearDefault(ctx context.Context, tenantID, productID uuid.UUID) error
}

type ImageRepository interface {
	Create(ctx context.Context, img *Image) error
	// GetByID and Delete are scoped by productID too — the caller already
	// verified that product belongs to the tenant, so the query enforces the
	// same boundary instead of trusting the image id alone.
	GetByID(ctx context.Context, productID, id uuid.UUID) (*Image, error)
	ListByProduct(ctx context.Context, productID uuid.UUID) ([]Image, error)
	Update(ctx context.Context, img *Image) error
	Delete(ctx context.Context, productID, id uuid.UUID) error
}

// CategoryLinkRepository manages the product <-> category join.
type CategoryLinkRepository interface {
	SetForProduct(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error
	ListCategoryIDs(ctx context.Context, productID uuid.UUID) ([]uuid.UUID, error)
}
