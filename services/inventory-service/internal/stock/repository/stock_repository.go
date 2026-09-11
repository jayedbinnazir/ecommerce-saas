// Package repository holds the database/sql implementations of the stock-module
// repositories.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/platform"
	"github.com/jayedbinnazir/inventory-service/internal/stock/domain"
)

type StockRepository struct {
	db platform.DBTX
}

func NewStockRepository(db platform.DBTX) *StockRepository {
	return &StockRepository{db: db}
}

var _ domain.StockRepository = (*StockRepository)(nil)

const stockColumns = `id, tenant_id, sku, on_hand, reserved, reorder_level, location, created_at, updated_at`

func scanStock(row platform.Scanner) (*domain.StockItem, error) {
	var s domain.StockItem
	if err := row.Scan(
		&s.ID, &s.TenantID, &s.SKU, &s.OnHand, &s.Reserved, &s.ReorderLevel, &s.Location, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StockRepository) Create(ctx context.Context, s *domain.StockItem) error {
	const q = `
		INSERT INTO stock_items (tenant_id, sku, reorder_level, location)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + stockColumns

	created, err := scanStock(r.db.QueryRowContext(ctx, q, s.TenantID, s.SKU, s.ReorderLevel, s.Location))
	if err != nil {
		if platform.IsUniqueViolation(err, "stock_items_tenant_sku_key") {
			return domain.ErrStockItemExists
		}
		return err
	}
	*s = *created
	return nil
}

func (r *StockRepository) GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.StockItem, error) {
	const q = `SELECT ` + stockColumns + ` FROM stock_items WHERE tenant_id = $1 AND sku = $2`
	s, err := scanStock(r.db.QueryRowContext(ctx, q, tenantID, sku))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrStockItemNotFound
	}
	return s, err
}

func (r *StockRepository) LockBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.StockItem, error) {
	const q = `SELECT ` + stockColumns + ` FROM stock_items WHERE tenant_id = $1 AND sku = $2 FOR UPDATE`
	s, err := scanStock(r.db.QueryRowContext(ctx, q, tenantID, sku))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrStockItemNotFound
	}
	return s, err
}

func (r *StockRepository) List(ctx context.Context, f domain.ListFilter) ([]domain.StockItem, error) {
	where := []string{"tenant_id = $1"}
	args := []any{f.TenantID}

	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		where = append(where, "sku::text LIKE $"+strconv.Itoa(len(args)))
	}
	if f.LowOnly {
		where = append(where, "reorder_level > 0 AND (on_hand - reserved) <= reorder_level")
	}

	args = append(args, f.Page.Limit, f.Page.Offset)
	q := `SELECT ` + stockColumns + ` FROM stock_items WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY sku LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.StockItem, 0)
	for rows.Next() {
		s, err := scanStock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *StockRepository) Update(ctx context.Context, s *domain.StockItem) error {
	const q = `
		UPDATE stock_items
		SET on_hand = $3, reserved = $4, reorder_level = $5, location = $6
		WHERE tenant_id = $1 AND sku = $2
		RETURNING ` + stockColumns

	updated, err := scanStock(r.db.QueryRowContext(ctx, q,
		s.TenantID, s.SKU, s.OnHand, s.Reserved, s.ReorderLevel, s.Location))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrStockItemNotFound
	}
	if err != nil {
		return err
	}
	*s = *updated
	return nil
}

func (r *StockRepository) Delete(ctx context.Context, tenantID uuid.UUID, sku string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM stock_items WHERE tenant_id = $1 AND sku = $2`, tenantID, sku)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrStockItemNotFound)
}
