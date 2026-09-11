package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Method is how the customer pays.
type Method string

const (
	MethodCard Method = "CARD" // Stripe PaymentIntent
	MethodCOD  Method = "COD"  // cash on delivery — collected when the order is fulfilled
)

func (m Method) Valid() bool { return m == MethodCard || m == MethodCOD }

// Status is the payment lifecycle.
type Status string

const (
	StatusPending  Status = "PENDING"  // CARD: awaiting the customer on Stripe; COD: awaiting delivery
	StatusCaptured Status = "CAPTURED" // money received
	StatusFailed   Status = "FAILED"   // Stripe declined / canceled
	StatusRefunded Status = "REFUNDED" // captured then reversed
)

// Payment is one payment attempt for one order. UNIQUE (tenant_id, order_id).
// Table "payments".
type Payment struct {
	ID            uuid.UUID  `json:"id"           db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"    db:"tenant_id"`
	OrderID       uuid.UUID  `json:"order_id"     db:"order_id"`
	CustomerID    uuid.UUID  `json:"customer_id"  db:"customer_id"`
	Method        Method     `json:"method"       db:"method"`
	Status        Status     `json:"status"       db:"status"`
	AmountCents   int64      `json:"amount_cents"   db:"amount_cents"`
	RefundedCents int64      `json:"refunded_cents" db:"refunded_cents"`
	Currency      string     `json:"currency"       db:"currency"`
	GatewayRef    *string    `json:"gateway_ref,omitempty"   db:"gateway_ref"`   // Stripe payment_intent id
	ClientSecret  *string    `json:"-"                       db:"client_secret"` // Stripe client_secret (frontend only)
	CreatedAt     time.Time  `json:"created_at"   db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"   db:"updated_at"`
	CapturedAt    *time.Time `json:"captured_at,omitempty"  db:"captured_at"`
	RefundedAt    *time.Time `json:"refunded_at,omitempty"  db:"refunded_at"`
}

// MarkCaptured moves the payment to CAPTURED and stamps the time.
func (p *Payment) MarkCaptured() {
	now := time.Now().UTC()
	p.Status = StatusCaptured
	p.CapturedAt = &now
}

// RefundableCents is how much of the payment can still be refunded.
func (p *Payment) RefundableCents() int64 { return p.AmountCents - p.RefundedCents }

// ApplyRefund books a refund of amount cents. When it reaches the full amount
// the status becomes REFUNDED.
func (p *Payment) ApplyRefund(amount int64) {
	p.RefundedCents += amount
	if p.RefundedCents >= p.AmountCents {
		now := time.Now().UTC()
		p.Status = StatusRefunded
		p.RefundedAt = &now
	}
}

type Repository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Payment, error)
	GetByOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*Payment, error)
	GetByGatewayRef(ctx context.Context, ref string) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
}
