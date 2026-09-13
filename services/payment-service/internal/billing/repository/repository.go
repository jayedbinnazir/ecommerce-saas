// Package repository holds the database/sql implementations of the billing repos.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/billing/domain"
	"github.com/jayedbinnazir/payment-service/internal/platform"
)

// ---- plans ----

type PlanRepository struct{ db platform.DBTX }

func NewPlanRepository(db platform.DBTX) *PlanRepository { return &PlanRepository{db: db} }

var _ domain.PlanRepository = (*PlanRepository)(nil)

const planColumns = `id, code, name, billing_interval, price_cents, currency, active, created_at, updated_at`

func scanPlan(row platform.Scanner) (*domain.Plan, error) {
	var p domain.Plan
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.Interval, &p.PriceCents, &p.Currency, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PlanRepository) List(ctx context.Context, activeOnly bool) ([]domain.Plan, error) {
	q := `SELECT ` + planColumns + ` FROM plans`
	if activeOnly {
		q += ` WHERE active`
	}
	q += ` ORDER BY price_cents`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Plan, 0)
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PlanRepository) GetByCode(ctx context.Context, code string) (*domain.Plan, error) {
	p, err := scanPlan(r.db.QueryRowContext(ctx, `SELECT `+planColumns+` FROM plans WHERE code = $1`, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPlanNotFound
	}
	return p, err
}

func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	p, err := scanPlan(r.db.QueryRowContext(ctx, `SELECT `+planColumns+` FROM plans WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPlanNotFound
	}
	return p, err
}

// ---- subscriptions ----

type SubscriptionRepository struct{ db platform.DBTX }

func NewSubscriptionRepository(db platform.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

var _ domain.SubscriptionRepository = (*SubscriptionRepository)(nil)

const subColumns = `id, user_id, plan_id, status, current_period_start, current_period_end, last_renewal_key, created_at, updated_at`

func scanSub(row platform.Scanner) (*domain.Subscription, error) {
	var s domain.Subscription
	err := row.Scan(&s.ID, &s.UserID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.LastRenewalKey, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// expireLapsed flips any of the user's ACTIVE rows whose period has already
// ended to EXPIRED. There is no cron/worker in this system, so expiry is
// evaluated lazily, on the next access, the same way IsEntitled already
// evaluated it incidentally (via current_period_end) -- this just makes the
// stored `status` column agree with that evaluation, and frees the
// subscriptions_one_active_per_user partial unique index so the user can
// renew.
func expireLapsed(ctx context.Context, db platform.DBTX, userID uuid.UUID) error {
	const q = `
		UPDATE subscriptions
		SET status = 'EXPIRED', updated_at = now()
		WHERE user_id = $1 AND status = 'ACTIVE' AND current_period_end <= now()`
	_, err := db.ExecContext(ctx, q, userID)
	return err
}

func (r *SubscriptionRepository) Create(ctx context.Context, s *domain.Subscription) error {
	const q = `
		INSERT INTO subscriptions (user_id, plan_id, status, current_period_start, current_period_end, last_renewal_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + subColumns

	created, err := scanSub(r.db.QueryRowContext(ctx, q,
		s.UserID, s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd, s.LastRenewalKey))
	if err != nil {
		if platform.IsUniqueViolation(err, "subscriptions_one_active_per_user") {
			return domain.ErrAlreadySubscribed
		}
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrPlanNotFound
		}
		return err
	}
	*s = *created
	return nil
}

func (r *SubscriptionRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	if err := expireLapsed(ctx, r.db, userID); err != nil {
		return nil, err
	}

	const q = `
		SELECT ` + subColumns + `
		FROM subscriptions
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY current_period_end DESC
		LIMIT 1`

	s, err := scanSub(r.db.QueryRowContext(ctx, q, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSubscriptionNotFound
	}
	return s, err
}

func (r *SubscriptionRepository) GetLatestByUserForUpdate(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	if err := expireLapsed(ctx, r.db, userID); err != nil {
		return nil, err
	}

	const q = `
		SELECT ` + subColumns + `
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE`

	s, err := scanSub(r.db.QueryRowContext(ctx, q, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrSubscriptionNotFound
	}
	return s, err
}

func (r *SubscriptionRepository) Update(ctx context.Context, s *domain.Subscription) error {
	const q = `
		UPDATE subscriptions
		SET status = $2, plan_id = $3, current_period_start = $4, current_period_end = $5, last_renewal_key = $6
		WHERE id = $1
		RETURNING ` + subColumns

	updated, err := scanSub(r.db.QueryRowContext(ctx, q,
		s.ID, s.Status, s.PlanID, s.CurrentPeriodStart, s.CurrentPeriodEnd, s.LastRenewalKey))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrSubscriptionNotFound
	}
	if err != nil {
		return err
	}
	*s = *updated
	return nil
}
