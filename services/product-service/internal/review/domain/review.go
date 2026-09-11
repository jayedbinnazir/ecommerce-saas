// Package domain holds the product-review aggregate.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
)

// Review is one customer's rating of one product. Table "product_reviews",
// UNIQUE (product_id, customer_id).
type Review struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	TenantID   uuid.UUID `json:"tenant_id"   db:"tenant_id"`
	ProductID  uuid.UUID `json:"product_id"  db:"product_id"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id"`
	Rating     int       `json:"rating"      db:"rating"` // 1-5
	Title      *string   `json:"title,omitempty" db:"title"`
	Body       *string   `json:"body,omitempty"  db:"body"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"  db:"updated_at"`
}

// Summary is the aggregate rating for a product.
type Summary struct {
	Average float64 `json:"average"` // 0 when there are no reviews
	Count   int     `json:"count"`
}

type Repository interface {
	// Upsert creates the review, or updates the caller's existing one.
	Upsert(ctx context.Context, r *Review) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Review, error)
	ListByProduct(ctx context.Context, productID uuid.UUID, page platform.Page) ([]Review, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Summary(ctx context.Context, productID uuid.UUID) (Summary, error)
	// SummaryByProducts returns the aggregate for many products in one query.
	SummaryByProducts(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID]Summary, error)
}
