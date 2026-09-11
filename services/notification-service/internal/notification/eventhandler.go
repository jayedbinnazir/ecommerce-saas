package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/notification-service/internal/events"
	"github.com/jayedbinnazir/notification-service/internal/notification/dto"
	"github.com/jayedbinnazir/notification-service/internal/notification/services"
)

// orderEvent / paymentEvent mirror the payloads order-service and payment-service
// publish (see their events packages).
type orderEvent struct {
	OrderID       string `json:"order_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	Status        string `json:"status"`
	Currency      string `json:"currency"`
	SubtotalCents int64  `json:"subtotal_cents"`
	ItemCount     int    `json:"item_count"`
	Refunded      bool   `json:"refunded"`
}

type paymentEvent struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	TenantID    string `json:"tenant_id"`
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Method      string `json:"method"`
}

// EventHandler turns an order/payment event into an in-app notification.
func EventHandler(svc *services.Service) events.Handler {
	return func(ctx context.Context, e events.Event) error {
		switch e.Type {
		case events.OrderPlaced, events.OrderConfirmed, events.OrderShipped, events.OrderCancelled:
			var oe orderEvent
			if err := e.Into(&oe); err != nil {
				return err
			}
			title, body := orderCopy(e.Type, oe)
			return create(ctx, svc, oe.CustomerID, oe.TenantID, e.Type, title, body, e.Data)

		case events.PaymentCaptured, events.PaymentRefunded:
			var pe paymentEvent
			if err := e.Into(&pe); err != nil {
				return err
			}
			title, body := paymentCopy(e.Type)
			return create(ctx, svc, pe.CustomerID, pe.TenantID, e.Type, title, body, e.Data)
		}
		return nil // event type we don't surface
	}
}

func create(ctx context.Context, svc *services.Service, userID, tenantID, typ, title, body string, data json.RawMessage) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	req := dto.CreateRequest{
		UserID: uid,
		Type:   typ,
		Title:  title,
		Body:   body,
	}
	if tenantID != "" {
		t := tenantID
		req.TenantID = &t
	}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &req.Data)
	}
	_, err = svc.Create(ctx, req)
	return err
}

func orderCopy(eventType string, oe orderEvent) (title, body string) {
	switch eventType {
	case events.OrderPlaced:
		return "Order placed", "We've received your order and are holding your items."
	case events.OrderConfirmed:
		return "Order confirmed", "Your order is confirmed and will be prepared for shipment."
	case events.OrderShipped:
		return "Order shipped", "Your order has been fulfilled and is on its way."
	case events.OrderCancelled:
		if oe.Refunded {
			return "Order cancelled", "Your order was cancelled and your payment refunded."
		}
		return "Order cancelled", "Your order was cancelled and the reserved items released."
	}
	return "Order update", "Your order status changed."
}

func paymentCopy(eventType string) (title, body string) {
	if eventType == events.PaymentRefunded {
		return "Payment refunded", "Your payment has been refunded."
	}
	return "Payment received", "We've received your payment. Thank you!"
}
