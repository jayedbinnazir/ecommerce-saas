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
	GetByID(ctx context.Context, id uuid.UUID) (*Variant, error)
	ListByProduct(ctx context.Context, productID uuid.UUID) ([]Variant, error)
	CountByProduct(ctx context.Context, productID uuid.UUID) (int, error)
	Update(ctx context.Context, v *Variant) error
	Delete(ctx context.Context, id uuid.UUID) error
	// ClearDefault unsets the current default variant for a product.
	ClearDefault(ctx context.Context, productID uuid.UUID) error
}

type ImageRepository interface {
	Create(ctx context.Context, img *Image) error
	GetByID(ctx context.Context, id uuid.UUID) (*Image, error)
	ListByProduct(ctx context.Context, productID uuid.UUID) ([]Image, error)
	Update(ctx context.Context, img *Image) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CategoryLinkRepository manages the product <-> category join.
type CategoryLinkRepository interface {
	SetForProduct(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error
	ListCategoryIDs(ctx context.Context, productID uuid.UUID) ([]uuid.UUID, error)
}
