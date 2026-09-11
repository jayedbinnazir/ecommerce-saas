package mail

import (
	"context"
	"fmt"
	"log"

	"github.com/jayedbinnazir/mail-service/internal/events"
	"github.com/jayedbinnazir/mail-service/internal/mail/dto"
	"github.com/jayedbinnazir/mail-service/internal/mail/services"
	"github.com/jayedbinnazir/mail-service/internal/userclient"
)

type orderEvent struct {
	OrderID         string `json:"order_id"`
	CustomerID      string `json:"customer_id"`
	Currency        string `json:"currency"`
	SubtotalCents   int64  `json:"subtotal_cents"`
	GrandTotalCents int64  `json:"grand_total_cents"`
	Status          string `json:"status"`
	Refunded        bool   `json:"refunded"`
}

type paymentEvent struct {
	OrderID     string `json:"order_id"`
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Method      string `json:"method"`
}

// templateFor maps an event type to the email template to send (or "" to skip).
func templateFor(eventType string) string {
	switch eventType {
	case events.OrderPlaced:
		return "order_confirmation"
	case events.OrderShipped:
		return "order_shipped"
	case events.OrderCancelled:
		return "order_cancelled"
	case events.PaymentCaptured:
		return "payment_receipt"
	default:
		return "" // order.confirmed, payment.refunded: no email in this slice
	}
}

// EventHandler renders and sends the transactional email for an order/payment event.
func EventHandler(svc *services.Service, users *userclient.Client) events.Handler {
	return func(ctx context.Context, e events.Event) error {
		template := templateFor(e.Type)
		if template == "" {
			return nil
		}

		customerID, currency, data := extract(e)
		if customerID == "" {
			return nil
		}

		if !users.Enabled() {
			log.Printf("[mail] no user-management configured; cannot email for %s", e.Type)
			return nil
		}
		u, err := users.Get(ctx, customerID)
		if err != nil {
			return err // userclient.ErrUpstream -> retried by the consumer
		}

		data["name"] = firstNonEmpty(u.Name, u.Email)
		data["store_name"] = "the store"
		data["currency"] = currency

		_, err = svc.Send(ctx, dto.SendRequest{To: u.Email, Template: template, Data: data})
		return err
	}
}

func extract(e events.Event) (customerID, currency string, data map[string]any) {
	data = map[string]any{}
	switch {
	case e.Type == events.PaymentCaptured || e.Type == events.PaymentRefunded:
		var pe paymentEvent
		if err := e.Into(&pe); err != nil {
			return "", "", data
		}
		data["order_number"] = shortID(pe.OrderID)
		data["amount"] = money(pe.AmountCents, pe.Currency)
		data["payment_method"] = pe.Method
		return pe.CustomerID, pe.Currency, data
	default:
		var oe orderEvent
		if err := e.Into(&oe); err != nil {
			return "", "", data
		}
		total := oe.GrandTotalCents
		if total == 0 {
			total = oe.SubtotalCents
		}
		data["order_number"] = shortID(oe.OrderID)
		data["total"] = money(total, oe.Currency)
		data["refunded"] = oe.Refunded
		return oe.CustomerID, oe.Currency, data
	}
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

func money(cents int64, currency string) string {
	return fmt.Sprintf("%s %d.%02d", currency, cents/100, cents%100)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
