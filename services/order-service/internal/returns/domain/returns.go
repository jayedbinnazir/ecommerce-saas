// Package domain holds the return-request aggregate.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/platform"
)

type Status string

const (
	StatusRequested Status = "REQUESTED" // customer asked, awaiting the seller
	StatusCompleted Status = "COMPLETED" // approved — refunded (+ optionally restocked)
	StatusRejected  Status = "REJECTED"  // seller declined
)

// Return is a customer's request to send items back. Table "order_returns".
type Return struct {
	ID             uuid.UUID  `json:"id"          db:"id"`
	OrderID        uuid.UUID  `json:"order_id"    db:"order_id"`
	TenantID       uuid.UUID  `json:"tenant_id"   db:"tenant_id"`
	CustomerID     uuid.UUID  `json:"customer_id" db:"customer_id"`
	Status         Status     `json:"status"      db:"status"`
	Reason         string     `json:"reason"      db:"reason"`
	ResolutionNote *string    `json:"resolution_note,omitempty" db:"resolution_note"`
	RefundCents    int64      `json:"refund_cents" db:"refund_cents"`
	Restocked      bool       `json:"restocked"   db:"restocked"`
	CreatedAt      time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"  db:"updated_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// Item is one line of a return. Table "order_return_items".
type Item struct {
	ID             uuid.UUID `json:"id"               db:"id"`
	ReturnID       uuid.UUID `json:"return_id"        db:"return_id"`
	SKU            string    `json:"sku"              db:"sku"`
	Quantity       int       `json:"quantity"         db:"quantity"`
	UnitPriceCents int64     `json:"unit_price_cents" db:"unit_price_cents"`
	ProductName    string    `json:"product_name"     db:"product_name"`
}

func (i Item) LineValueCents() int64 { return int64(i.Quantity) * i.UnitPriceCents }

// Detail is a return with its lines.
type Detail struct {
	Return
	Items []Item
}

type ListFilter struct {
	TenantID uuid.UUID
	Status   *Status
	Page     platform.Page
}

type Repository interface {
	Create(ctx context.Context, r *Return, items []Item) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Return, error)
	ListItems(ctx context.Context, returnID uuid.UUID) ([]Item, error)
	ListByOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]Return, error)
	List(ctx context.Context, f ListFilter) ([]Return, error)
	// ItemsByReturnIDs fetches items for many returns in one query (no N+1).
	ItemsByReturnIDs(ctx context.Context, returnIDs []uuid.UUID) (map[uuid.UUID][]Item, error)
	// AlreadyReturnedBySKU sums, per SKU, the quantity already covered by a
	// non-rejected return (REQUESTED or COMPLETED) on this order — one query,
	// used so a new return request can't push the total past what was ordered.
	AlreadyReturnedBySKU(ctx context.Context, orderID uuid.UUID) (map[string]int, error)
	Resolve(ctx context.Context, tenantID, id uuid.UUID, status Status, note *string, refundCents int64, restocked bool) error
}
