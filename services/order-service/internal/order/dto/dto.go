// Package dto holds the request/response payloads for the order module.
package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/order/domain"
)

// ---- requests ----

// AddressInput is a delivery / billing address supplied at checkout. It is
// snapshotted onto the order verbatim.
type AddressInput struct {
	RecipientName string  `json:"recipient_name" binding:"required,max=255"`
	Phone         string  `json:"phone" binding:"required,max=32"`
	AddressLine1  string  `json:"address_line_1" binding:"required,max=255"`
	AddressLine2  *string `json:"address_line_2" binding:"omitempty,max=255"`
	City          string  `json:"city" binding:"required,max=128"`
	State         *string `json:"state" binding:"omitempty,max=128"`
	PostalCode    *string `json:"postal_code" binding:"omitempty,max=32"`
	Country       string  `json:"country" binding:"required,len=2,uppercase"`
}

// CheckoutRequest turns the caller's active cart into an order.
type CheckoutRequest struct {
	ShippingAddress AddressInput  `json:"shipping_address" binding:"required"`
	BillingAddress  *AddressInput `json:"billing_address"` // nil = same as shipping
	ShippingRateID  string        `json:"shipping_rate_id" binding:"required,uuid"`
	CouponCode      *string       `json:"coupon_code" binding:"omitempty,max=64"`
}

// JSON marshals an address for storage; never fails for a validated struct.
func (a AddressInput) JSON() json.RawMessage {
	b, _ := json.Marshal(a)
	return b
}

// PayRequest chooses how to pay: CARD (Stripe) or COD.
type PayRequest struct {
	PaymentMethod *string `json:"payment_method" binding:"omitempty,max=40"`
}

// FulfilRequest carries the shipment tracking info (all optional).
type FulfilRequest struct {
	Carrier        *string `json:"carrier" binding:"omitempty,max=64"`
	TrackingNumber *string `json:"tracking_number" binding:"omitempty,max=128"`
}

// ---- responses ----

type OrderItemResponse struct {
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

func fromItem(it *domain.Item) OrderItemResponse {
	return OrderItemResponse{
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

type OrderResponse struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Status     string    `json:"status"`
	Currency   string    `json:"currency"`

	SubtotalCents   int64   `json:"subtotal_cents"`
	DiscountCents   int64   `json:"discount_cents"`
	ShippingCents   int64   `json:"shipping_cents"`
	TaxCents        int64   `json:"tax_cents"`
	GrandTotalCents int64   `json:"grand_total_cents"`
	ItemCount       int     `json:"item_count"`
	CouponCode      *string `json:"coupon_code,omitempty"`

	ShippingAddress json.RawMessage `json:"shipping_address,omitempty"`
	BillingAddress  json.RawMessage `json:"billing_address,omitempty"`

	PaymentMethod   *string    `json:"payment_method,omitempty"`
	PaymentStatus   string     `json:"payment_status"`
	PaymentID       *uuid.UUID `json:"payment_id,omitempty"`
	TrackingCarrier *string    `json:"tracking_carrier,omitempty"`
	TrackingNumber  *string    `json:"tracking_number,omitempty"`
	// ClientSecret is set only on the Pay response for a CARD payment awaiting
	// the customer in Stripe.js.
	ClientSecret *string             `json:"client_secret,omitempty"`
	Items        []OrderItemResponse `json:"items"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	PaidAt       *time.Time          `json:"paid_at,omitempty"`
	FulfilledAt  *time.Time          `json:"fulfilled_at,omitempty"`
	CancelledAt  *time.Time          `json:"cancelled_at,omitempty"`
}

func FromDetail(d *domain.Detail) OrderResponse {
	items := make([]OrderItemResponse, len(d.Items))
	for i := range d.Items {
		items[i] = fromItem(&d.Items[i])
	}
	return OrderResponse{
		ID:              d.ID,
		TenantID:        d.TenantID,
		CustomerID:      d.CustomerID,
		Status:          string(d.Status),
		Currency:        d.Currency,
		SubtotalCents:   d.SubtotalCents,
		DiscountCents:   d.DiscountCents,
		ShippingCents:   d.ShippingCents,
		TaxCents:        d.TaxCents,
		GrandTotalCents: d.GrandTotalCents,
		ItemCount:       d.ItemCount,
		CouponCode:      d.CouponCode,
		ShippingAddress: d.ShippingAddress,
		BillingAddress:  d.BillingAddress,
		PaymentMethod:   d.PaymentMethod,
		PaymentStatus:   d.PaymentStatus,
		PaymentID:       d.PaymentID,
		TrackingCarrier: d.TrackingCarrier,
		TrackingNumber:  d.TrackingNumber,
		Items:           items,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		PaidAt:          d.PaidAt,
		FulfilledAt:     d.FulfilledAt,
		CancelledAt:     d.CancelledAt,
	}
}

// FromPayResult renders the Pay response (order + one-time client secret).
func FromPayResult(d *domain.Detail, clientSecret *string) OrderResponse {
	resp := FromDetail(d)
	resp.ClientSecret = clientSecret
	return resp
}

func FromDetails(ds []domain.Detail) []OrderResponse {
	out := make([]OrderResponse, len(ds))
	for i := range ds {
		out[i] = FromDetail(&ds[i])
	}
	return out
}
