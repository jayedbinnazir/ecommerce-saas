// Package order holds the Kafka event handler that keeps an order's status in
// sync with payment-service's webhook-driven captures/failures. Without this,
// an order only ever advances when a client explicitly calls POST .../pay
// again; a card captured (or declined) by a Stripe webhook while nobody is
// looking would otherwise sit unresolved forever.
package order

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/events"
	"github.com/jayedbinnazir/order-service/internal/order/services"
)

// paymentEvent mirrors the payload payment-service publishes (see its own
// internal/events and internal/payment/services/service.go:publish).
type paymentEvent struct {
	OrderID  string `json:"order_id"`
	TenantID string `json:"tenant_id"`
}

// EventHandler applies payment.captured / payment.failed events to the
// matching order. Both target service methods are idempotent, so retried or
// out-of-order delivery is safe.
func EventHandler(svc *services.Service) events.Handler {
	return func(ctx context.Context, e events.Event) error {
		if e.Type != events.PaymentCaptured && e.Type != events.PaymentFailed {
			return nil // order.*, payment.refunded: nothing for order-service to apply
		}

		var pe paymentEvent
		if err := e.Into(&pe); err != nil {
			log.Printf("[order] bad %s payload, skipping: %v", e.Type, err)
			return nil // malformed message — retrying won't fix it
		}
		tenantID, err := uuid.Parse(pe.TenantID)
		if err != nil {
			log.Printf("[order] bad tenant_id on %s, skipping: %v", e.Type, err)
			return nil
		}
		orderID, err := uuid.Parse(pe.OrderID)
		if err != nil {
			log.Printf("[order] bad order_id on %s, skipping: %v", e.Type, err)
			return nil
		}

		if e.Type == events.PaymentCaptured {
			return svc.ApplyPaymentCaptured(ctx, tenantID, orderID)
		}
		return svc.ApplyPaymentFailed(ctx, tenantID, orderID)
	}
}
