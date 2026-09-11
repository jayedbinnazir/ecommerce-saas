// Package dto holds the request/response payloads for the stock module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/stock/domain"
)

// ---- requests ----

// CreateStockItemRequest registers a SKU so its stock can be tracked. Quantities
// start at zero; use the receive/adjust operations to change them.
type CreateStockItemRequest struct {
	SKU          string  `json:"sku" binding:"required,max=100"`
	ReorderLevel int     `json:"reorder_level" binding:"gte=0"`
	Location     *string `json:"location" binding:"omitempty,max=255"`
}

type UpdateStockItemRequest struct {
	ReorderLevel *int    `json:"reorder_level" binding:"omitempty,gte=0"`
	Location     *string `json:"location" binding:"omitempty,max=255"` // "" clears it
}

func (r UpdateStockItemRequest) IsEmpty() bool {
	return r.ReorderLevel == nil && r.Location == nil
}

// ReceiveRequest adds stock that has arrived.
type ReceiveRequest struct {
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
	Reason    *string `json:"reason" binding:"omitempty,max=500"`
	Reference *string `json:"reference" binding:"omitempty,max=255"`
}

// AdjustRequest applies a manual correction; quantity may be negative.
type AdjustRequest struct {
	Quantity int     `json:"quantity" binding:"required"`
	Reason   *string `json:"reason" binding:"omitempty,max=500"`
}

// MoveRequest is the body for the per-SKU reserve / release / ship routes.
type MoveRequest struct {
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
	Reference *string `json:"reference" binding:"omitempty,max=255"`
}

// BulkLine is one SKU + quantity in a bulk move.
type BulkLine struct {
	SKU      string `json:"sku" binding:"required,max=100"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

// BulkMoveRequest is the body for the internal bulk reserve / release / ship
// endpoints used by order-service at checkout. All lines succeed or none do.
type BulkMoveRequest struct {
	Reference *string    `json:"reference" binding:"omitempty,max=255"`
	Items     []BulkLine `json:"items" binding:"required,min=1,max=200,dive"`
}

type BulkMoveResponse struct {
	Items []StockItemResponse `json:"items"`
}

// ---- responses ----

type StockItemResponse struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	SKU          string    `json:"sku"`
	OnHand       int       `json:"on_hand"`
	Reserved     int       `json:"reserved"`
	Available    int       `json:"available"`
	ReorderLevel int       `json:"reorder_level"`
	LowStock     bool      `json:"low_stock"`
	Location     *string   `json:"location,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func FromStockItem(s *domain.StockItem) StockItemResponse {
	return StockItemResponse{
		ID:           s.ID,
		TenantID:     s.TenantID,
		SKU:          s.SKU,
		OnHand:       s.OnHand,
		Reserved:     s.Reserved,
		Available:    s.Available(),
		ReorderLevel: s.ReorderLevel,
		LowStock:     s.LowStock(),
		Location:     s.Location,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func FromStockItems(items []domain.StockItem) []StockItemResponse {
	out := make([]StockItemResponse, len(items))
	for i := range items {
		out[i] = FromStockItem(&items[i])
	}
	return out
}

// AvailabilityResponse is the public storefront view: no internal counts.
type AvailabilityResponse struct {
	SKU       string `json:"sku"`
	Available int    `json:"available"`
	InStock   bool   `json:"in_stock"`
}

func Availability(s *domain.StockItem) AvailabilityResponse {
	return AvailabilityResponse{
		SKU:       s.SKU,
		Available: s.Available(),
		InStock:   s.Available() > 0,
	}
}

type MovementResponse struct {
	ID            uuid.UUID  `json:"id"`
	SKU           string     `json:"sku"`
	Type          string     `json:"type"`
	Quantity      int        `json:"quantity"`
	OnHandAfter   int        `json:"on_hand_after"`
	ReservedAfter int        `json:"reserved_after"`
	Reason        *string    `json:"reason,omitempty"`
	Reference     *string    `json:"reference,omitempty"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func FromMovement(m *domain.Movement) MovementResponse {
	return MovementResponse{
		ID:            m.ID,
		SKU:           m.SKU,
		Type:          string(m.Type),
		Quantity:      m.Quantity,
		OnHandAfter:   m.OnHandAfter,
		ReservedAfter: m.ReservedAfter,
		Reason:        m.Reason,
		Reference:     m.Reference,
		CreatedBy:     m.CreatedBy,
		CreatedAt:     m.CreatedAt,
	}
}

func FromMovements(ms []domain.Movement) []MovementResponse {
	out := make([]MovementResponse, len(ms))
	for i := range ms {
		out[i] = FromMovement(&ms[i])
	}
	return out
}
