package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/cart-service/internal/cart/domain"
	"github.com/jayedbinnazir/cart-service/internal/platform"
)

type ItemRepository struct {
	db platform.DBTX
}

func NewItemRepository(db platform.DBTX) *ItemRepository {
	return &ItemRepository{db: db}
}

var _ domain.ItemRepository = (*ItemRepository)(nil)

const itemColumns = `
	id, cart_id, product_id, sku, quantity, unit_price_cents,
	currency, product_name, variant_title, created_at, updated_at`

func scanItem(row platform.Scanner) (*domain.Item, error) {
	var it domain.Item
	err := row.Scan(
		&it.ID, &it.CartID, &it.ProductID, &it.SKU, &it.Quantity, &it.UnitPriceCents,
		&it.Currency, &it.ProductName, &it.VariantTitle, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *ItemRepository) ListByCart(ctx context.Context, cartID uuid.UUID) ([]domain.Item, error) {
	const q = `SELECT ` + itemColumns + ` FROM cart_items WHERE cart_id = $1 ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Item, 0)
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

func (r *ItemRepository) GetBySKU(ctx context.Context, cartID uuid.UUID, sku string) (*domain.Item, error) {
	const q = `SELECT ` + itemColumns + ` FROM cart_items WHERE cart_id = $1 AND sku = $2`
	it, err := scanItem(r.db.QueryRowContext(ctx, q, cartID, sku))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	return it, err
}

func (r *ItemRepository) Create(ctx context.Context, it *domain.Item) error {
	const q = `
		INSERT INTO cart_items (
			cart_id, product_id, sku, quantity, unit_price_cents,
			currency, product_name, variant_title
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + itemColumns

	created, err := scanItem(r.db.QueryRowContext(ctx, q,
		it.CartID, it.ProductID, it.SKU, it.Quantity, it.UnitPriceCents,
		it.Currency, it.ProductName, it.VariantTitle,
	))
	if err != nil {
		return err
	}
	*it = *created
	return nil
}

func (r *ItemRepository) SetQuantity(ctx context.Context, id uuid.UUID, quantity int) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE cart_items SET quantity = $2 WHERE id = $1`, id, quantity)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrItemNotFound)
}

func (r *ItemRepository) Delete(ctx context.Context, cartID uuid.UUID, sku string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM cart_items WHERE cart_id = $1 AND sku = $2`, cartID, sku)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrItemNotFound)
}

func (r *ItemRepository) DeleteAllByCart(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	return err
}
