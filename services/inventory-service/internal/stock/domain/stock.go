package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/platform"
)

// StockItem is the stock level for one SKU inside one tenant. The SKU is a plain
// string shared with product-service (a product variant's sku); there is no
// foreign key across services.
//
// Table "stock_items", UNIQUE (tenant_id, sku). Invariant: reserved <= on_hand.
type StockItem struct {
	ID           uuid.UUID `json:"id"            db:"id"`
	TenantID     uuid.UUID `json:"tenant_id"     db:"tenant_id"`
	SKU          string    `json:"sku"           db:"sku"`
	OnHand       int       `json:"on_hand"       db:"on_hand"`       // physically in the warehouse
	Reserved     int       `json:"reserved"      db:"reserved"`      // allocated to open orders/carts
	ReorderLevel int       `json:"reorder_level" db:"reorder_level"` // low-stock threshold, 0 = disabled
	Location     *string   `json:"location,omitempty" db:"location"` // free-text warehouse/bin
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}

// Available is what a customer can still buy: on hand minus already reserved.
func (s StockItem) Available() int { return s.OnHand - s.Reserved }

// LowStock reports whether available stock has fallen to the reorder level.
func (s StockItem) LowStock() bool { return s.ReorderLevel > 0 && s.Available() <= s.ReorderLevel }

// MovementType is the kind of change a Movement records.
type MovementType string

const (
	MovementReceive MovementType = "RECEIVE" // stock arrived: on_hand += qty
	MovementShip    MovementType = "SHIP"    // stock left for a shipped order: on_hand -= qty, reserved -= qty
	MovementAdjust  MovementType = "ADJUST"  // manual correction: on_hand += qty (qty may be negative)
	MovementReserve MovementType = "RESERVE" // held for an order: reserved += qty
	MovementRelease MovementType = "RELEASE" // reservation cancelled: reserved -= qty
)

func (t MovementType) Valid() bool {
	switch t {
	case MovementReceive, MovementShip, MovementAdjust, MovementReserve, MovementRelease:
		return true
	default:
		return false
	}
}

// Movement is one append-only entry in a SKU's stock ledger.
// Table "stock_movements".
type Movement struct {
	ID            uuid.UUID    `json:"id"              db:"id"`
	StockItemID   uuid.UUID    `json:"stock_item_id"   db:"stock_item_id"`
	TenantID      uuid.UUID    `json:"tenant_id"       db:"tenant_id"`
	SKU           string       `json:"sku"             db:"sku"`
	Type          MovementType `json:"type"            db:"type"`
	Quantity      int          `json:"quantity"        db:"quantity"`       // as requested; ADJUST may be negative
	OnHandAfter   int          `json:"on_hand_after"   db:"on_hand_after"`  // snapshot after applying
	ReservedAfter int          `json:"reserved_after"  db:"reserved_after"` // snapshot after applying
	Reason        *string      `json:"reason,omitempty"    db:"reason"`
	Reference     *string      `json:"reference,omitempty" db:"reference"` // external ref: order id, PO number...
	CreatedBy     *uuid.UUID   `json:"created_by,omitempty" db:"created_by"`
	CreatedAt     time.Time    `json:"created_at"      db:"created_at"`
}

// ListFilter narrows a stock-item listing.
type ListFilter struct {
	TenantID uuid.UUID
	Search   string // matches sku, empty = any
	LowOnly  bool   // only items at or below their reorder level
	Page     platform.Page
}

type StockRepository interface {
	Create(ctx context.Context, s *StockItem) error
	GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*StockItem, error)
	// LockBySKU reads the row FOR UPDATE inside a transaction.
	LockBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*StockItem, error)
	List(ctx context.Context, f ListFilter) ([]StockItem, error)
	Update(ctx context.Context, s *StockItem) error
	Delete(ctx context.Context, tenantID uuid.UUID, sku string) error
}

type MovementRepository interface {
	Create(ctx context.Context, m *Movement) error
	ListByItem(ctx context.Context, stockItemID uuid.UUID, page platform.Page) ([]Movement, error)
}
