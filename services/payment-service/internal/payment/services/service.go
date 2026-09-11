// Package services holds the payment-module logic: create a payment for an order
// (Stripe PaymentIntent for CARD, a pending record for COD), then capture / settle
// / refund it.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/authz"
	"github.com/jayedbinnazir/payment-service/internal/events"
	"github.com/jayedbinnazir/payment-service/internal/gateway"
	"github.com/jayedbinnazir/payment-service/internal/payment/domain"
	"github.com/jayedbinnazir/payment-service/internal/payment/dto"
)

// ErrGateway wraps any failure talking to Stripe. httpx maps it to 502.
var ErrGateway = errors.New("payment gateway error")

type Service struct {
	repo   domain.Repository
	gw     gateway.Gateway
	events *events.Publisher
	authz  *authz.Client
}

func New(repo domain.Repository, gw gateway.Gateway, publisher *events.Publisher, authzClient *authz.Client) *Service {
	return &Service{repo: repo, gw: gw, events: publisher, authz: authzClient}
}

// publish emits a payment event consumed by notification-service (in-app) and
// mail-service (receipt / refund email).
func (s *Service) publish(ctx context.Context, p *domain.Payment, eventType string) {
	s.events.Publish(ctx, events.TopicPayments, p.OrderID.String(), eventType, map[string]any{
		"payment_id":     p.ID.String(),
		"order_id":       p.OrderID.String(),
		"tenant_id":      p.TenantID.String(),
		"customer_id":    p.CustomerID.String(),
		"amount_cents":   p.AmountCents,
		"refunded_cents": p.RefundedCents,
		"currency":       p.Currency,
		"method":         string(p.Method),
	})
}

// CreateForOrder is idempotent on (tenant, order): a second call returns the
// existing payment. CARD opens a Stripe PaymentIntent; COD records a pending
// payment to be collected on delivery.
func (s *Service) CreateForOrder(ctx context.Context, tenantID uuid.UUID, req dto.CreatePaymentRequest) (*domain.Payment, error) {
	method := domain.Method(strings.ToUpper(req.Method))
	if !method.Valid() {
		return nil, domain.ErrInvalidMethod
	}
	if req.AmountCents <= 0 {
		return nil, domain.ErrInvalidAmount
	}

	if existing, err := s.repo.GetByOrder(ctx, tenantID, req.OrderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrPaymentNotFound) {
		return nil, err
	}

	p := &domain.Payment{
		ID:          uuid.New(),
		TenantID:    tenantID,
		OrderID:     req.OrderID,
		CustomerID:  req.CustomerID,
		Method:      method,
		Status:      domain.StatusPending,
		AmountCents: req.AmountCents,
		Currency:    strings.ToUpper(req.Currency),
	}

	if method == domain.MethodCard {
		intent, err := s.gw.CreateIntent(ctx, p.AmountCents, p.Currency, p.OrderID.String())
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrGateway, err)
		}
		p.GatewayRef = &intent.ID
		p.ClientSecret = &intent.ClientSecret
		if intent.Status == gateway.StatusSucceeded {
			p.MarkCaptured()
		}
	}

	if err := s.repo.Create(ctx, p); err != nil {
		if errors.Is(err, domain.ErrPaymentExists) {
			return s.repo.GetByOrder(ctx, tenantID, req.OrderID)
		}
		return nil, err
	}
	if p.Status == domain.StatusCaptured {
		s.publish(ctx, p, events.PaymentCaptured)
	}
	return p, nil
}

// Sync refreshes a pending CARD payment from Stripe.
func (s *Service) Sync(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p.Method != domain.MethodCard || p.Status != domain.StatusPending || p.GatewayRef == nil {
		return p, nil
	}

	intent, err := s.gw.GetIntent(ctx, *p.GatewayRef)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGateway, err)
	}
	switch intent.Status {
	case gateway.StatusSucceeded:
		p.MarkCaptured()
		if err := s.repo.Update(ctx, p); err != nil {
			return nil, err
		}
		s.publish(ctx, p, events.PaymentCaptured)
	case gateway.StatusFailed:
		p.Status = domain.StatusFailed
		if err := s.repo.Update(ctx, p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// Settle marks a pending COD payment collected (order fulfilled / delivered).
func (s *Service) Settle(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p.Method != domain.MethodCOD || p.Status != domain.StatusPending {
		return nil, domain.ErrNotSettleable
	}
	p.MarkCaptured()
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.publish(ctx, p, events.PaymentCaptured)
	return p, nil
}

// Refund reverses a captured payment. amountCents nil = the full remaining
// amount; a smaller value is a partial refund (used for returns), after which
// the payment stays CAPTURED with refunded_cents > 0.
func (s *Service) Refund(ctx context.Context, tenantID, id uuid.UUID, amountCents *int64) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.StatusCaptured {
		return nil, domain.ErrNotRefundable
	}

	amount := p.RefundableCents()
	if amountCents != nil {
		amount = *amountCents
	}
	if amount <= 0 || amount > p.RefundableCents() {
		return nil, domain.ErrInvalidAmount
	}

	if p.Method == domain.MethodCard && p.GatewayRef != nil {
		if err := s.gw.Refund(ctx, *p.GatewayRef, &amount); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrGateway, err)
		}
	}

	p.ApplyRefund(amount)
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.publish(ctx, p, events.PaymentRefunded)
	return p, nil
}

// HandleWebhook advances a payment from a verified Stripe event. Unknown events
// and payments we don't own are ignored.
func (s *Service) HandleWebhook(ctx context.Context, eventType, intentID string) error {
	p, err := s.repo.GetByGatewayRef(ctx, intentID)
	if errors.Is(err, domain.ErrPaymentNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if p.Status != domain.StatusPending {
		return nil
	}
	switch eventType {
	case "payment_intent.succeeded":
		p.MarkCaptured()
		if err := s.repo.Update(ctx, p); err != nil {
			return err
		}
		s.publish(ctx, p, events.PaymentCaptured)
		return nil
	case "payment_intent.payment_failed", "payment_intent.canceled":
		p.Status = domain.StatusFailed
		return s.repo.Update(ctx, p)
	}
	return nil
}

// GetForOrder / Get return a payment the caller is allowed to see (owner, or a
// tenant ADMIN/MANAGER).
func (s *Service) GetForOrder(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID) (*domain.Payment, error) {
	p, err := s.repo.GetByOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	return s.authorize(ctx, p, customerID, rawToken)
}

func (s *Service) Get(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, id uuid.UUID) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.authorize(ctx, p, customerID, rawToken)
}

func (s *Service) authorize(ctx context.Context, p *domain.Payment, customerID uuid.UUID, rawToken string) (*domain.Payment, error) {
	if p.CustomerID == customerID {
		return p, nil
	}
	role, err := s.authz.RoleInTenant(ctx, rawToken, p.TenantID)
	if err != nil {
		return nil, err
	}
	switch role {
	case authz.RoleAdmin, authz.RoleManager, authz.RoleSuperAdmin:
		return p, nil
	default:
		return nil, domain.ErrNotOwner
	}
}
