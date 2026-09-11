package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Status is the lifecycle state of a cart.
type Status string

const (
	StatusActive    Status = "ACTIVE"    // the shopper is still filling it
	StatusOrdered   Status = "ORDERED"   // converted to an order (order-service)
	StatusAbandoned Status = "ABANDONED" // given up on / expired
)

// Cart is one shopper's basket inside one tenant. At most one ACTIVE cart per
// (tenant_id, customer_id). customer_id is the user id from the access token.
// Table "carts".
type Cart struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	TenantID   uuid.UUID `json:"tenant_id"   db:"tenant_id"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id"`
	Status     Status    `json:"status"      db:"status"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"  db:"updated_at"`
}

// Item is one line in a cart. Price and names are snapshotted when the line is
// first added so the total stays stable even if the catalog changes.
// Table "cart_items", UNIQUE (cart_id, sku).
type Item struct {
	ID             uuid.UUID `json:"id"               db:"id"`
	CartID         uuid.UUID `json:"cart_id"          db:"cart_id"`
	ProductID      uuid.UUID `json:"product_id"       db:"product_id"`
	SKU            string    `json:"sku"              db:"sku"`
	Quantity       int       `json:"quantity"         db:"quantity"`
	UnitPriceCents int64     `json:"unit_price_cents" db:"unit_price_cents"`
	Currency       string    `json:"currency"         db:"currency"`
	ProductName    string    `json:"product_name"     db:"product_name"`
	VariantTitle   *string   `json:"variant_title,omitempty" db:"variant_title"`
	CreatedAt      time.Time `json:"created_at"       db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"       db:"updated_at"`
}

// SubtotalCents is quantity × unit price.
func (i Item) SubtotalCents() int64 { return int64(i.Quantity) * i.UnitPriceCents }

// Detail is a cart with its lines and derived totals.
type Detail struct {
	Cart
	Items []Item
}

// SubtotalCents is the sum of every line subtotal.
func (d Detail) SubtotalCents() int64 {
	var total int64
	for _, it := range d.Items {
		total += it.SubtotalCents()
	}
	return total
}

// ItemCount is the total number of units across all lines.
func (d Detail) ItemCount() int {
	n := 0
	for _, it := range d.Items {
		n += it.Quantity
	}
	return n
}

// Currency returns the cart currency (all lines share one), or "" when empty.
func (d Detail) Currency() string {
	if len(d.Items) == 0 {
		return ""
	}
	return d.Items[0].Currency
}

type CartRepository interface {
	GetActive(ctx context.Context, tenantID, customerID uuid.UUID) (*Cart, error)
	Create(ctx context.Context, c *Cart) error
	// Touch bumps updated_at so an idle-cart sweep can find stale carts.
	Touch(ctx context.Context, cartID uuid.UUID) error
}

type ItemRepository interface {
	ListByCart(ctx context.Context, cartID uuid.UUID) ([]Item, error)
	GetBySKU(ctx context.Context, cartID uuid.UUID, sku string) (*Item, error)
	Create(ctx context.Context, it *Item) error
	SetQuantity(ctx context.Context, id uuid.UUID, quantity int) error
	Delete(ctx context.Context, cartID uuid.UUID, sku string) error
	DeleteAllByCart(ctx context.Context, cartID uuid.UUID) error
}
