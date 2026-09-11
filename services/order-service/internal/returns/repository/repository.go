// Package repository is the database/sql implementation of the returns repo.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/order-service/internal/platform"
	"github.com/jayedbinnazir/order-service/internal/returns/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const returnColumns = `
	id, order_id, tenant_id, customer_id, status, reason, resolution_note,
	refund_cents, restocked, created_at, updated_at, resolved_at`

func scanReturn(row platform.Scanner) (*domain.Return, error) {
	var r domain.Return
	err := row.Scan(
		&r.ID, &r.OrderID, &r.TenantID, &r.CustomerID, &r.Status, &r.Reason, &r.ResolutionNote,
		&r.RefundCents, &r.Restocked, &r.CreatedAt, &r.UpdatedAt, &r.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const itemColumns = `id, return_id, sku, quantity, unit_price_cents, product_name`

func scanItem(row platform.Scanner) (*domain.Item, error) {
	var it domain.Item
	if err := row.Scan(&it.ID, &it.ReturnID, &it.SKU, &it.Quantity, &it.UnitPriceCents, &it.ProductName); err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *Repository) Create(ctx context.Context, ret *domain.Return, items []domain.Item) error {
	return platform.RunInTx(ctx, r.db, func(tx *sql.Tx) error {
		const q = `
			INSERT INTO order_returns (order_id, tenant_id, customer_id, status, reason, refund_cents)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING ` + returnColumns
		created, err := scanReturn(tx.QueryRowContext(ctx, q,
			ret.OrderID, ret.TenantID, ret.CustomerID, ret.Status, ret.Reason, ret.RefundCents))
		if err != nil {
			return err
		}
		*ret = *created

		if len(items) == 0 {
			return nil
		}
		rows := make([]string, len(items))
		args := make([]any, 0, len(items)*5)
		for i, it := range items {
			n := i * 5
			rows[i] = fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)", n+1, n+2, n+3, n+4, n+5)
			args = append(args, ret.ID, it.SKU, it.Quantity, it.UnitPriceCents, it.ProductName)
		}
		ins := `INSERT INTO order_return_items (return_id, sku, quantity, unit_price_cents, product_name) VALUES ` +
			strings.Join(rows, ",")
		_, err = tx.ExecContext(ctx, ins, args...)
		return err
	})
}

func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Return, error) {
	const q = `SELECT ` + returnColumns + ` FROM order_returns WHERE tenant_id = $1 AND id = $2`
	ret, err := scanReturn(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReturnNotFound
	}
	return ret, err
}

func (r *Repository) ListItems(ctx context.Context, returnID uuid.UUID) ([]domain.Item, error) {
	const q = `SELECT ` + itemColumns + ` FROM order_return_items WHERE return_id = $1 ORDER BY id`
	rows, err := r.db.QueryContext(ctx, q, returnID)
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

func (r *Repository) ListByOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]domain.Return, error) {
	const q = `SELECT ` + returnColumns + ` FROM order_returns WHERE tenant_id = $1 AND order_id = $2 ORDER BY created_at DESC`
	return r.query(ctx, q, tenantID, orderID)
}

func (r *Repository) List(ctx context.Context, f domain.ListFilter) ([]domain.Return, error) {
	where := []string{"tenant_id = $1"}
	args := []any{f.TenantID}
	if f.Status != nil {
		args = append(args, *f.Status)
		where = append(where, "status = $"+strconv.Itoa(len(args)))
	}
	args = append(args, f.Page.Limit, f.Page.Offset)
	q := `SELECT ` + returnColumns + ` FROM order_returns WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	return r.query(ctx, q, args...)
}

func (r *Repository) query(ctx context.Context, q string, args ...any) ([]domain.Return, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Return, 0)
	for rows.Next() {
		ret, err := scanReturn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ret)
	}
	return out, rows.Err()
}

func (r *Repository) ItemsByReturnIDs(ctx context.Context, returnIDs []uuid.UUID) (map[uuid.UUID][]domain.Item, error) {
	out := make(map[uuid.UUID][]domain.Item, len(returnIDs))
	if len(returnIDs) == 0 {
		return out, nil
	}
	ids := make([]string, len(returnIDs))
	for i, id := range returnIDs {
		ids[i] = id.String()
	}
	const q = `SELECT ` + itemColumns + ` FROM order_return_items WHERE return_id = ANY($1) ORDER BY return_id, id`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out[it.ReturnID] = append(out[it.ReturnID], *it)
	}
	return out, rows.Err()
}

func (r *Repository) Resolve(ctx context.Context, id uuid.UUID, status domain.Status, note *string, refundCents int64, restocked bool) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE order_returns
		SET status = $2, resolution_note = $3, refund_cents = $4, restocked = $5, resolved_at = now()
		WHERE id = $1 AND status = 'REQUESTED'`, id, status, note, refundCents, restocked)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrAlreadyResolved)
}
