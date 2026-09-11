// Package repository is the database/sql implementation of the order repository.
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

	"github.com/jayedbinnazir/order-service/internal/order/domain"
	"github.com/jayedbinnazir/order-service/internal/platform"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const orderColumns = `
	id, tenant_id, customer_id, status, currency,
	subtotal_cents, discount_cents, shipping_cents, tax_cents, grand_total_cents, item_count,
	coupon_code, shipping_address, billing_address, idempotency_key,
	payment_method, payment_status, tracking_carrier, tracking_number, payment_id,
	created_at, updated_at, paid_at, fulfilled_at, cancelled_at`

func scanOrder(row platform.Scanner) (*domain.Order, error) {
	var o domain.Order
	var shipAddr, billAddr []byte
	err := row.Scan(
		&o.ID, &o.TenantID, &o.CustomerID, &o.Status, &o.Currency,
		&o.SubtotalCents, &o.DiscountCents, &o.ShippingCents, &o.TaxCents, &o.GrandTotalCents, &o.ItemCount,
		&o.CouponCode, &shipAddr, &billAddr, &o.IdempotencyKey,
		&o.PaymentMethod, &o.PaymentStatus, &o.TrackingCarrier, &o.TrackingNumber, &o.PaymentID,
		&o.CreatedAt, &o.UpdatedAt, &o.PaidAt, &o.FulfilledAt, &o.CancelledAt,
	)
	if err != nil {
		return nil, err
	}
	o.ShippingAddress = shipAddr
	o.BillingAddress = billAddr
	return &o, nil
}

const itemColumns = `
	id, order_id, product_id, sku, quantity, unit_price_cents,
	currency, product_name, variant_title, created_at`

func scanItem(row platform.Scanner) (*domain.Item, error) {
	var it domain.Item
	err := row.Scan(
		&it.ID, &it.OrderID, &it.ProductID, &it.SKU, &it.Quantity, &it.UnitPriceCents,
		&it.Currency, &it.ProductName, &it.VariantTitle, &it.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// Create inserts the order and every line in one transaction. Items go in with a
// single multi-row INSERT — never one statement per line.
func (r *Repository) Create(ctx context.Context, o *domain.Order, items []domain.Item) error {
	return platform.RunInTx(ctx, r.db, func(tx *sql.Tx) error {
		const q = `
			INSERT INTO orders (
				id, tenant_id, customer_id, status, currency,
				subtotal_cents, discount_cents, shipping_cents, tax_cents, grand_total_cents, item_count,
				coupon_code, shipping_address, billing_address, idempotency_key
			)
			VALUES ($1,$2,$3,$4,$5, $6,$7,$8,$9,$10,$11, $12,$13,$14,$15)
			RETURNING ` + orderColumns

		created, err := scanOrder(tx.QueryRowContext(ctx, q,
			o.ID, o.TenantID, o.CustomerID, o.Status, o.Currency,
			o.SubtotalCents, o.DiscountCents, o.ShippingCents, o.TaxCents, o.GrandTotalCents, o.ItemCount,
			o.CouponCode, nullableJSON(o.ShippingAddress), nullableJSON(o.BillingAddress), o.IdempotencyKey))
		if err != nil {
			return err
		}
		*o = *created

		if len(items) == 0 {
			return nil
		}

		valueRows := make([]string, len(items))
		args := make([]any, 0, len(items)*8)
		for i, it := range items {
			n := i * 8
			valueRows[i] = fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8)
			args = append(args, o.ID, it.ProductID, it.SKU, it.Quantity,
				it.UnitPriceCents, it.Currency, it.ProductName, it.VariantTitle)
		}
		ins := `
			INSERT INTO order_items
				(order_id, product_id, sku, quantity, unit_price_cents, currency, product_name, variant_title)
			VALUES ` + strings.Join(valueRows, ",")
		_, err = tx.ExecContext(ctx, ins, args...)
		return err
	})
}

func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Order, error) {
	const q = `SELECT ` + orderColumns + ` FROM orders WHERE tenant_id = $1 AND id = $2`
	o, err := scanOrder(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	return o, err
}

func (r *Repository) GetByIdempotencyKey(ctx context.Context, tenantID, customerID uuid.UUID, key string) (*domain.Order, error) {
	const q = `SELECT ` + orderColumns + ` FROM orders
	           WHERE tenant_id = $1 AND customer_id = $2 AND idempotency_key = $3`
	o, err := scanOrder(r.db.QueryRowContext(ctx, q, tenantID, customerID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	return o, err
}

// nullableJSON turns an empty raw message into a SQL NULL so the JSONB column
// stays null rather than storing an empty string.
func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}

func (r *Repository) ListItems(ctx context.Context, orderID uuid.UUID) ([]domain.Item, error) {
	const q = `SELECT ` + itemColumns + ` FROM order_items WHERE order_id = $1 ORDER BY created_at, id`
	rows, err := r.db.QueryContext(ctx, q, orderID)
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

func (r *Repository) List(ctx context.Context, f domain.ListFilter) ([]domain.Order, error) {
	where := []string{"tenant_id = $1"}
	args := []any{f.TenantID}

	if f.CustomerID != nil {
		args = append(args, *f.CustomerID)
		where = append(where, "customer_id = $"+strconv.Itoa(len(args)))
	}
	if f.Status != nil {
		args = append(args, *f.Status)
		where = append(where, "status = $"+strconv.Itoa(len(args)))
	}

	args = append(args, f.Page.Limit, f.Page.Offset)
	q := `SELECT ` + orderColumns + ` FROM orders WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

// ItemsByOrderIDs loads every line for the given orders in ONE query.
func (r *Repository) ItemsByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]domain.Item, error) {
	out := make(map[uuid.UUID][]domain.Item, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}

	ids := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		ids[i] = id.String()
	}

	const q = `
		SELECT ` + itemColumns + `
		FROM order_items
		WHERE order_id = ANY($1)
		ORDER BY order_id, created_at, id`
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
		out[it.OrderID] = append(out[it.OrderID], *it)
	}
	return out, rows.Err()
}

func (r *Repository) AttachPayment(ctx context.Context, id, paymentID uuid.UUID, method domain.Method) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET payment_id = $2, payment_method = $3 WHERE id = $1`, id, paymentID, string(method))
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrOrderNotFound)
}

func (r *Repository) MarkConfirmed(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = 'CONFIRMED'
		WHERE id = $1 AND status = 'PENDING_PAYMENT'`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrInvalidTransition)
}

func (r *Repository) MarkPaymentPaid(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET payment_status = 'PAID', paid_at = now()
		WHERE id = $1 AND payment_status = 'PENDING'`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrInvalidTransition)
}

func (r *Repository) MarkPaymentRefunded(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET payment_status = 'REFUNDED'
		WHERE id = $1 AND payment_status = 'PAID'`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrInvalidTransition)
}

func (r *Repository) MarkFulfilled(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'FULFILLED', fulfilled_at = now(),
		    tracking_carrier = $2, tracking_number = $3
		WHERE id = $1 AND status = 'CONFIRMED'`, id, carrier, trackingNumber)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrInvalidTransition)
}

func (r *Repository) MarkCancelled(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = 'CANCELLED', cancelled_at = now()
		WHERE id = $1 AND status IN ('PENDING_PAYMENT', 'CONFIRMED')`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrInvalidTransition)
}
