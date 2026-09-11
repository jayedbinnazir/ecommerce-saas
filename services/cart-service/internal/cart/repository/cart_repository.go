// Package repository holds the database/sql implementations of the cart-module
// repositories.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/cart-service/internal/cart/domain"
	"github.com/jayedbinnazir/cart-service/internal/platform"
)

type CartRepository struct {
	db platform.DBTX
}

func NewCartRepository(db platform.DBTX) *CartRepository {
	return &CartRepository{db: db}
}

var _ domain.CartRepository = (*CartRepository)(nil)

const cartColumns = `id, tenant_id, customer_id, status, created_at, updated_at`

func scanCart(row platform.Scanner) (*domain.Cart, error) {
	var c domain.Cart
	if err := row.Scan(&c.ID, &c.TenantID, &c.CustomerID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CartRepository) GetActive(ctx context.Context, tenantID, customerID uuid.UUID) (*domain.Cart, error) {
	const q = `
		SELECT ` + cartColumns + `
		FROM carts
		WHERE tenant_id = $1 AND customer_id = $2 AND status = 'ACTIVE'`
	c, err := scanCart(r.db.QueryRowContext(ctx, q, tenantID, customerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCartNotFound
	}
	return c, err
}

func (r *CartRepository) Create(ctx context.Context, c *domain.Cart) error {
	const q = `
		INSERT INTO carts (tenant_id, customer_id, status)
		VALUES ($1, $2, $3)
		RETURNING ` + cartColumns
	created, err := scanCart(r.db.QueryRowContext(ctx, q, c.TenantID, c.CustomerID, c.Status))
	if err != nil {
		return err
	}
	*c = *created
	return nil
}

func (r *CartRepository) Touch(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE carts SET updated_at = now() WHERE id = $1`, cartID)
	return err
}
