// Package dto holds the request/response payloads for the cart module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/cart-service/internal/cart/domain"
)

// ---- requests ----

type AddItemRequest struct {
	ProductID string `json:"product_id" binding:"required,uuid"`
	SKU       string `json:"sku" binding:"required,max=100"`
	Quantity  int    `json:"quantity" binding:"required,gt=0"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

// ---- responses ----

type CartItemResponse struct {
	ID             uuid.UUID `json:"id"`
	ProductID      uuid.UUID `json:"product_id"`
	SKU            string    `json:"sku"`
	Quantity       int       `json:"quantity"`
	UnitPriceCents int64     `json:"unit_price_cents"`
	Currency       string    `json:"currency"`
	ProductName    string    `json:"product_name"`
	VariantTitle   *string   `json:"variant_title,omitempty"`
	SubtotalCents  int64     `json:"subtotal_cents"`
}

func fromItem(it *domain.Item) CartItemResponse {
	return CartItemResponse{
		ID:             it.ID,
		ProductID:      it.ProductID,
		SKU:            it.SKU,
		Quantity:       it.Quantity,
		UnitPriceCents: it.UnitPriceCents,
		Currency:       it.Currency,
		ProductName:    it.ProductName,
		VariantTitle:   it.VariantTitle,
		SubtotalCents:  it.SubtotalCents(),
	}
}

type CartResponse struct {
	ID            uuid.UUID          `json:"id"`
	TenantID      uuid.UUID          `json:"tenant_id"`
	CustomerID    uuid.UUID          `json:"customer_id"`
	Status        string             `json:"status"`
	Currency      string             `json:"currency"` // "" while the cart is empty
	Items         []CartItemResponse `json:"items"`
	ItemCount     int                `json:"item_count"`
	SubtotalCents int64              `json:"subtotal_cents"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func FromDetail(d *domain.Detail) CartResponse {
	items := make([]CartItemResponse, len(d.Items))
	for i := range d.Items {
		items[i] = fromItem(&d.Items[i])
	}
	return CartResponse{
		ID:            d.ID,
		TenantID:      d.TenantID,
		CustomerID:    d.CustomerID,
		Status:        string(d.Status),
		Currency:      d.Currency(),
		Items:         items,
		ItemCount:     d.ItemCount(),
		SubtotalCents: d.SubtotalCents(),
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
