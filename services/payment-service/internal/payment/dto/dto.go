// Package dto holds the request/response payloads for the payment module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/payment/domain"
)

// CreatePaymentRequest is the internal body order-service posts at checkout / pay.
// tenant_id comes from the path, not the body.
type CreatePaymentRequest struct {
	OrderID     uuid.UUID `json:"order_id" binding:"required"`
	CustomerID  uuid.UUID `json:"customer_id" binding:"required"`
	AmountCents int64     `json:"amount_cents" binding:"required,gt=0"`
	Currency    string    `json:"currency" binding:"required,len=3"`
	Method      string    `json:"method" binding:"required,oneof=CARD COD"`
}

type PaymentResponse struct {
	ID            uuid.UUID `json:"id"`
	OrderID       uuid.UUID `json:"order_id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Method        string    `json:"method"`
	Status        string    `json:"status"`
	AmountCents   int64     `json:"amount_cents"`
	RefundedCents int64     `json:"refunded_cents"`
	Currency      string    `json:"currency"`
	// ClientSecret is set only for CARD payments — the frontend uses it with
	// Stripe.js to collect the card. It is never persisted in responses after.
	ClientSecret *string    `json:"client_secret,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CapturedAt   *time.Time `json:"captured_at,omitempty"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty"`
}

// FromPayment renders a payment. includeSecret is true only on the create call.
func FromPayment(p *domain.Payment, includeSecret bool) PaymentResponse {
	resp := PaymentResponse{
		ID:            p.ID,
		OrderID:       p.OrderID,
		TenantID:      p.TenantID,
		CustomerID:    p.CustomerID,
		Method:        string(p.Method),
		Status:        string(p.Status),
		AmountCents:   p.AmountCents,
		RefundedCents: p.RefundedCents,
		Currency:      p.Currency,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		CapturedAt:    p.CapturedAt,
		RefundedAt:    p.RefundedAt,
	}
	if includeSecret {
		resp.ClientSecret = p.ClientSecret
	}
	return resp
}
