package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Interval string

const (
	IntervalMonth Interval = "MONTH"
	IntervalYear  Interval = "YEAR"
)

// AddTo returns t advanced by one billing interval.
func (iv Interval) AddTo(t time.Time) time.Time {
	switch iv {
	case IntervalYear:
		return t.AddDate(1, 0, 0)
	default:
		return t.AddDate(0, 1, 0)
	}
}

type SubStatus string

const (
	SubActive   SubStatus = "ACTIVE"
	SubCanceled SubStatus = "CANCELED"
	SubExpired  SubStatus = "EXPIRED"
	SubPastDue  SubStatus = "PAST_DUE"
)

// Plan is a purchasable subscription tier. Table "plans".
type Plan struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	Code       string    `json:"code"        db:"code"` // UNIQUE, e.g. "starter-monthly"
	Name       string    `json:"name"        db:"name"`
	Interval   Interval  `json:"interval"    db:"billing_interval"`
	PriceCents int64     `json:"price_cents" db:"price_cents"`
	Currency   string    `json:"currency"    db:"currency"`
	Active     bool      `json:"active"      db:"active"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"  db:"updated_at"`
}

// Subscription is a user's current entitlement to run stores. Table
// "subscriptions"; one ACTIVE row per user (partial unique index).
type Subscription struct {
	ID                 uuid.UUID `json:"id"          db:"id"`
	UserID             uuid.UUID `json:"user_id"     db:"user_id"`
	PlanID             uuid.UUID `json:"plan_id"     db:"plan_id"`
	Status             SubStatus `json:"status"      db:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"   db:"current_period_end"`
	// LastRenewalKey is the most recently applied renewal Idempotency-Key (if
	// any), so a retried/duplicate renewal request is a no-op instead of
	// extending the period twice.
	LastRenewalKey *string   `json:"-"           db:"last_renewal_key"`
	CreatedAt      time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"  db:"updated_at"`
}

// IsEntitled reports whether the subscription currently grants access. Only
// ACTIVE-and-within-period counts — CANCELED, EXPIRED and PAST_DUE (a failed
// renewal charge's grace state) all fall through to false, which is exactly
// the "no active subscription -> can't create a tenant" gate.
func (s *Subscription) IsEntitled(now time.Time) bool {
	return s.Status == SubActive && now.Before(s.CurrentPeriodEnd)
}

type PlanRepository interface {
	List(ctx context.Context, activeOnly bool) ([]Plan, error)
	GetByCode(ctx context.Context, code string) (*Plan, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *Subscription) error
	GetActiveByUser(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	// GetLatestByUserForUpdate returns the user's most recent subscription
	// row regardless of status, row-locked (FOR UPDATE) so renewal/reactivation
	// read-check-write is atomic under concurrent duplicate requests. Before
	// reading, it lazily flips any ACTIVE row whose period has lapsed to
	// EXPIRED (there is no scheduler in this system; expiry is evaluated
	// on access, the same way IsEntitled already evaluated it incidentally).
	GetLatestByUserForUpdate(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	Update(ctx context.Context, s *Subscription) error
}
