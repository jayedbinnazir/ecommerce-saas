// Package services holds the payment-module logic: create a payment for an order
// (Stripe PaymentIntent for CARD, a pending record for COD), then capture / settle
// / refund it.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/authz"
	"github.com/jayedbinnazir/payment-service/internal/events"
	"github.com/jayedbinnazir/payment-service/internal/gateway"
	"github.com/jayedbinnazir/payment-service/internal/payment/domain"
	"github.com/jayedbinnazir/payment-service/internal/payment/dto"
	"github.com/jayedbinnazir/payment-service/internal/payment/repository"
	"github.com/jayedbinnazir/payment-service/internal/platform"
)

// ErrGateway wraps any failure talking to Stripe. httpx maps it to 502.
var ErrGateway = errors.New("payment gateway error")

type Service struct {
	db     *sql.DB // held so HandleWebhook can run inside a locked transaction
	repo   domain.Repository
	gw     gateway.Gateway
	events *events.Publisher
	authz  *authz.Client
}

func New(db *sql.DB, repo domain.Repository, gw gateway.Gateway, publisher *events.Publisher, authzClient *authz.Client) *Service {
	return &Service{db: db, repo: repo, gw: gw, events: publisher, authz: authzClient}
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

// Cancel voids a payment that hasn't captured yet — used when an order is
// cancelled while still PENDING_PAYMENT, so a delayed Stripe capture can't
// land after the fact. Only a PENDING payment can be cancelled (a CAPTURED one
// must go through Refund instead — this never touches money that already moved).
// If Stripe rejects the void (e.g. it already captured a moment earlier), the
// payment is left PENDING and the error is returned: the caller (order-service)
// treats this as best-effort and the eventual capture webhook will detect the
// order is cancelled and refund it instead (see order/services.ApplyPaymentCaptured).
func (s *Service) Cancel(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.StatusPending {
		return nil, domain.ErrNotCancellable
	}

	if p.Method == domain.MethodCard && p.GatewayRef != nil {
		if err := s.gw.Cancel(ctx, *p.GatewayRef); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrGateway, err)
		}
	}

	p.Status = domain.StatusFailed
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
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
// HandleWebhook applies a verified Stripe event to the matching payment.
//
// The read-check-write is wrapped in one DB transaction with the payment row
// locked (SELECT ... FOR UPDATE), so two genuinely concurrent deliveries of the
// same event (Stripe retries, or the same event hitting two replicas) can't
// both pass the "still PENDING" guard and both capture/publish — the second
// transaction blocks on the lock, then sees the first's committed change and
// no-ops. The Kafka publish happens only after the transaction commits, and
// only on the delivery that actually made the change, so it can't double-fire.
func (s *Service) HandleWebhook(ctx context.Context, eventType, intentID string) error {
	var (
		captured, failed bool
		p                *domain.Payment
	)
	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		txRepo := repository.New(tx)

		loaded, err := txRepo.GetByGatewayRefForUpdate(ctx, intentID)
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return nil // unknown/foreign PaymentIntent — ignore, ack the webhook
		}
		if err != nil {
			return err
		}
		if loaded.Status != domain.StatusPending {
			return nil // already processed (this event or a later one) — no-op
		}

		switch eventType {
		case "payment_intent.succeeded":
			loaded.MarkCaptured()
			if err := txRepo.Update(ctx, loaded); err != nil {
				return err
			}
			captured = true
		case "payment_intent.payment_failed", "payment_intent.canceled":
			loaded.Status = domain.StatusFailed
			if err := txRepo.Update(ctx, loaded); err != nil {
				return err
			}
			failed = true
		}
		p = loaded
		return nil
	})
	if err != nil {
		return err
	}

	switch {
	case captured:
		s.publish(ctx, p, events.PaymentCaptured)
	case failed:
		s.publish(ctx, p, events.PaymentFailed)
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
