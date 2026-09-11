// Package dto holds request/response payloads for the returns module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/returns/domain"
)

// ---- requests ----

type ReturnItemInput struct {
	SKU      string `json:"sku" binding:"required,max=100"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

type RequestReturnRequest struct {
	Reason string            `json:"reason" binding:"required,max=1000"`
	Items  []ReturnItemInput `json:"items" binding:"required,min=1,max=100,dive"`
}

// ResolveReturnRequest approves or rejects a return. On approval RefundCents
// defaults to the value of the returned items; Restock defaults to true.
type ResolveReturnRequest struct {
	Approve     bool    `json:"approve"`
	RefundCents *int64  `json:"refund_cents" binding:"omitempty,gte=0"`
	Restock     *bool   `json:"restock"`
	Note        *string `json:"note" binding:"omitempty,max=1000"`
}

// ---- responses ----

type ReturnItemResponse struct {
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	ProductName    string `json:"product_name"`
	LineValueCents int64  `json:"line_value_cents"`
}

type ReturnResponse struct {
	ID             uuid.UUID            `json:"id"`
	OrderID        uuid.UUID            `json:"order_id"`
	CustomerID     uuid.UUID            `json:"customer_id"`
	Status         string               `json:"status"`
	Reason         string               `json:"reason"`
	ResolutionNote *string              `json:"resolution_note,omitempty"`
	RefundCents    int64                `json:"refund_cents"`
	Restocked      bool                 `json:"restocked"`
	Items          []ReturnItemResponse `json:"items"`
	CreatedAt      time.Time            `json:"created_at"`
	ResolvedAt     *time.Time           `json:"resolved_at,omitempty"`
}

func FromDetail(d *domain.Detail) ReturnResponse {
	items := make([]ReturnItemResponse, len(d.Items))
	for i := range d.Items {
		it := d.Items[i]
		items[i] = ReturnItemResponse{
			SKU: it.SKU, Quantity: it.Quantity, UnitPriceCents: it.UnitPriceCents,
			ProductName: it.ProductName, LineValueCents: it.LineValueCents(),
		}
	}
	return ReturnResponse{
		ID:             d.ID,
		OrderID:        d.OrderID,
		CustomerID:     d.CustomerID,
		Status:         string(d.Status),
		Reason:         d.Reason,
		ResolutionNote: d.ResolutionNote,
		RefundCents:    d.RefundCents,
		Restocked:      d.Restocked,
		Items:          items,
		CreatedAt:      d.CreatedAt,
		ResolvedAt:     d.ResolvedAt,
	}
}

func FromDetails(ds []domain.Detail) []ReturnResponse {
	out := make([]ReturnResponse, len(ds))
	for i := range ds {
		out[i] = FromDetail(&ds[i])
	}
	return out
}
