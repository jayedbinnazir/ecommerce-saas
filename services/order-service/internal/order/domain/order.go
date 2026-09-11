package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/platform"
)

// Status is the fulfilment lifecycle of an order (separate from payment).
type Status string

const (
	StatusPendingPayment Status = "PENDING_PAYMENT" // created, stock reserved, not yet paid/confirmed
	StatusConfirmed      Status = "CONFIRMED"       // card captured OR cash-on-delivery accepted — ok to fulfil
	StatusFulfilled      Status = "FULFILLED"       // shipped — reserved stock removed from on-hand
	StatusCancelled      Status = "CANCELLED"       // cancelled — reserved stock released
)

// PaymentStatus mirrors payment-service for this order.
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "PENDING"  // awaiting card capture, or COD not yet collected
	PaymentPaid     PaymentStatus = "PAID"     // money received
	PaymentRefunded PaymentStatus = "REFUNDED" // captured then reversed
	PaymentFailed   PaymentStatus = "FAILED"
)

// Method is how the customer chose to pay.
type Method string

const (
	MethodCard Method = "CARD"
	MethodCOD  Method = "COD"
)

func (m Method) Valid() bool { return m == MethodCard || m == MethodCOD }

// Order is an immutable snapshot of a checkout. Table "orders".
type Order struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	TenantID   uuid.UUID `json:"tenant_id"   db:"tenant_id"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id"`
	Status     Status    `json:"status"      db:"status"`
	Currency   string    `json:"currency"    db:"currency"`

	// money breakdown, all in cents; grand_total = subtotal - discount + shipping + tax
	SubtotalCents   int64 `json:"subtotal_cents"    db:"subtotal_cents"`
	DiscountCents   int64 `json:"discount_cents"    db:"discount_cents"`
	ShippingCents   int64 `json:"shipping_cents"    db:"shipping_cents"`
	TaxCents        int64 `json:"tax_cents"         db:"tax_cents"`
	GrandTotalCents int64 `json:"grand_total_cents" db:"grand_total_cents"`
	ItemCount       int   `json:"item_count"        db:"item_count"`

	CouponCode      *string         `json:"coupon_code,omitempty"      db:"coupon_code"`
	ShippingAddress json.RawMessage `json:"shipping_address,omitempty" db:"shipping_address"`
	BillingAddress  json.RawMessage `json:"billing_address,omitempty"  db:"billing_address"`
	IdempotencyKey  *string         `json:"-"                          db:"idempotency_key"`

	PaymentMethod   *string `json:"payment_method,omitempty" db:"payment_method"`
	PaymentStatus   string  `json:"payment_status" db:"payment_status"`
	TrackingCarrier *string `json:"tracking_carrier,omitempty" db:"tracking_carrier"`
	TrackingNumber  *string `json:"tracking_number,omitempty" db:"tracking_number"`

	PaymentID   *uuid.UUID `json:"payment_id,omitempty" db:"payment_id"`
	CreatedAt   time.Time  `json:"created_at"     db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"     db:"updated_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"      db:"paid_at"`
	FulfilledAt *time.Time `json:"fulfilled_at,omitempty" db:"fulfilled_at"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
}

// Item is one line of an order, copied from the cart at checkout.
type Item struct {
	ID             uuid.UUID `json:"id"               db:"id"`
	OrderID        uuid.UUID `json:"order_id"         db:"order_id"`
	ProductID      uuid.UUID `json:"product_id"       db:"product_id"`
	SKU            string    `json:"sku"              db:"sku"`
	Quantity       int       `json:"quantity"         db:"quantity"`
	UnitPriceCents int64     `json:"unit_price_cents" db:"unit_price_cents"`
	Currency       string    `json:"currency"         db:"currency"`
	ProductName    string    `json:"product_name"     db:"product_name"`
	VariantTitle   *string   `json:"variant_title,omitempty" db:"variant_title"`
	CreatedAt      time.Time `json:"created_at"       db:"created_at"`
}

func (i Item) SubtotalCents() int64 { return int64(i.Quantity) * i.UnitPriceCents }

// Detail is an order with its lines.
type Detail struct {
	Order
	Items []Item
}

// ListFilter narrows an order listing.
type ListFilter struct {
	TenantID   uuid.UUID
	CustomerID *uuid.UUID
	Status     *Status
	Page       platform.Page
}

type Repository interface {
	Create(ctx context.Context, o *Order, items []Item) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Order, error)
	// GetByIdempotencyKey returns a prior checkout for the same key, or
	// ErrOrderNotFound. Used to make POST /orders safe to retry.
	GetByIdempotencyKey(ctx context.Context, tenantID, customerID uuid.UUID, key string) (*Order, error)
	ListItems(ctx context.Context, orderID uuid.UUID) ([]Item, error)
	List(ctx context.Context, f ListFilter) ([]Order, error)
	ItemsByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]Item, error)

	// AttachPayment records which payment-service payment covers this order.
	AttachPayment(ctx context.Context, id, paymentID uuid.UUID, method Method) error
	MarkConfirmed(ctx context.Context, id uuid.UUID) error
	MarkPaymentPaid(ctx context.Context, id uuid.UUID) error
	MarkPaymentRefunded(ctx context.Context, id uuid.UUID) error
	MarkFulfilled(ctx context.Context, id uuid.UUID, carrier, trackingNumber *string) error
	MarkCancelled(ctx context.Context, id uuid.UUID) error
}
