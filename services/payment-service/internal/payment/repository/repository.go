// Package repository is the database/sql implementation of the payment repository.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/payment/domain"
	"github.com/jayedbinnazir/payment-service/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `
	id, tenant_id, order_id, customer_id, method, status, amount_cents, refunded_cents, currency,
	gateway_ref, client_secret, created_at, updated_at, captured_at, refunded_at`

func scan(row platform.Scanner) (*domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(
		&p.ID, &p.TenantID, &p.OrderID, &p.CustomerID, &p.Method, &p.Status, &p.AmountCents, &p.RefundedCents, &p.Currency,
		&p.GatewayRef, &p.ClientSecret, &p.CreatedAt, &p.UpdatedAt, &p.CapturedAt, &p.RefundedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, p *domain.Payment) error {
	const q = `
		INSERT INTO payments (
			id, tenant_id, order_id, customer_id, method, status, amount_cents, currency,
			gateway_ref, client_secret, captured_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + columns

	created, err := scan(r.db.QueryRowContext(ctx, q,
		p.ID, p.TenantID, p.OrderID, p.CustomerID, p.Method, p.Status, p.AmountCents, p.Currency,
		p.GatewayRef, p.ClientSecret, p.CapturedAt,
	))
	if err != nil {
		if platform.IsUniqueViolation(err, "payments_tenant_order_key") {
			return domain.ErrPaymentExists // caller re-reads by order
		}
		return err
	}
	*p = *created
	return nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	const q = `SELECT ` + columns + ` FROM payments WHERE tenant_id = $1 AND id = $2`
	p, err := scan(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	return p, err
}

func (r *Repository) GetByOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Payment, error) {
	const q = `SELECT ` + columns + ` FROM payments WHERE tenant_id = $1 AND order_id = $2`
	p, err := scan(r.db.QueryRowContext(ctx, q, tenantID, orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	return p, err
}

func (r *Repository) GetByGatewayRef(ctx context.Context, ref string) (*domain.Payment, error) {
	const q = `SELECT ` + columns + ` FROM payments WHERE gateway_ref = $1`
	p, err := scan(r.db.QueryRowContext(ctx, q, ref))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	return p, err
}

// GetByGatewayRefForUpdate is GetByGatewayRef with a row lock, so a webhook
// handler can read-check-write the payment atomically inside a transaction —
// concurrent deliveries of the same Stripe event serialize on this row instead
// of racing past the same "already processed" check. gateway_ref has a unique
// index (where not null), so this locks at most one row.
func (r *Repository) GetByGatewayRefForUpdate(ctx context.Context, ref string) (*domain.Payment, error) {
	const q = `SELECT ` + columns + ` FROM payments WHERE gateway_ref = $1 FOR UPDATE`
	p, err := scan(r.db.QueryRowContext(ctx, q, ref))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	return p, err
}

func (r *Repository) Update(ctx context.Context, p *domain.Payment) error {
	const q = `
		UPDATE payments
		SET status = $3, gateway_ref = $4, client_secret = $5,
		    captured_at = $6, refunded_at = $7, refunded_cents = $8
		WHERE id = $1 AND tenant_id = $2
		RETURNING ` + columns

	updated, err := scan(r.db.QueryRowContext(ctx, q,
		p.ID, p.TenantID, p.Status, p.GatewayRef, p.ClientSecret, p.CapturedAt, p.RefundedAt, p.RefundedCents))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrPaymentNotFound
	}
	if err != nil {
		return err
	}
	*p = *updated
	return nil
}
