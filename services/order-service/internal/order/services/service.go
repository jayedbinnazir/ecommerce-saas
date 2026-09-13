// Package services holds the order-module application logic: checkout
// orchestration (cart -> reserve stock -> create order) and the payment /
// fulfilment / cancellation transitions.
package services

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/cartclient"
	"github.com/jayedbinnazir/order-service/internal/events"
	"github.com/jayedbinnazir/order-service/internal/inventoryclient"
	"github.com/jayedbinnazir/order-service/internal/order/domain"
	"github.com/jayedbinnazir/order-service/internal/order/dto"
	"github.com/jayedbinnazir/order-service/internal/order/repository"
	"github.com/jayedbinnazir/order-service/internal/paymentclient"
	"github.com/jayedbinnazir/order-service/internal/platform"
	scdomain "github.com/jayedbinnazir/order-service/internal/storeconfig/domain"
	storeconfigrepo "github.com/jayedbinnazir/order-service/internal/storeconfig/repository"
	storeconfig "github.com/jayedbinnazir/order-service/internal/storeconfig/services"
)

type Service struct {
	repo        *repository.Repository
	cart        *cartclient.Client
	inventory   *inventoryclient.Client
	payments    *paymentclient.Client
	storeconfig *storeconfig.Service
	events      *events.Publisher
	authz       *authz.Client
}

func New(repo *repository.Repository, cart *cartclient.Client, inventory *inventoryclient.Client, payments *paymentclient.Client, cfg *storeconfig.Service, publisher *events.Publisher, authzClient *authz.Client) *Service {
	return &Service{repo: repo, cart: cart, inventory: inventory, payments: payments, storeconfig: cfg, events: publisher, authz: authzClient}
}

// publishOrder emits an order-lifecycle event. The customer's notification and
// email come from consumers (notification-service, mail-service).
func (s *Service) publishOrder(ctx context.Context, o *domain.Order, eventType string) {
	s.events.Publish(ctx, events.TopicOrders, o.ID.String(), eventType, orderEventData(o, nil))
}

func orderEventData(o *domain.Order, extra map[string]any) map[string]any {
	d := map[string]any{
		"order_id":          o.ID.String(),
		"tenant_id":         o.TenantID.String(),
		"customer_id":       o.CustomerID.String(),
		"status":            string(o.Status),
		"currency":          o.Currency,
		"subtotal_cents":    o.SubtotalCents,
		"discount_cents":    o.DiscountCents,
		"shipping_cents":    o.ShippingCents,
		"tax_cents":         o.TaxCents,
		"grand_total_cents": o.GrandTotalCents,
		"item_count":        o.ItemCount,
	}
	if o.TrackingCarrier != nil {
		d["tracking_carrier"] = *o.TrackingCarrier
	}
	if o.TrackingNumber != nil {
		d["tracking_number"] = *o.TrackingNumber
	}
	for k, v := range extra {
		d[k] = v
	}
	return d
}

// PayResult is a paid order plus the one-time Stripe client secret (CARD, when
// the payment still needs the customer to complete it in the browser).
type PayResult struct {
	Detail       *domain.Detail
	ClientSecret *string
}

// ---------------------------------------------------------------------
// Checkout
// ---------------------------------------------------------------------

// Checkout turns the caller's active cart into an order: reads the cart, prices
// it (shipping + discount + tax), reserves stock, writes the order, redeems the
// coupon, then clears the cart. Passing the same idempotencyKey again returns the
// order that first call created (created=false), so POST /orders is retry-safe.
func (s *Service) Checkout(ctx context.Context, tenantID, customerID uuid.UUID, rawToken, idempotencyKey string, req dto.CheckoutRequest) (detail *domain.Detail, created bool, err error) {
	if idempotencyKey != "" {
		if existing, gerr := s.repo.GetByIdempotencyKey(ctx, tenantID, customerID, idempotencyKey); gerr == nil {
			d, derr := s.detail(ctx, tenantID, existing.ID)
			return d, false, derr
		} else if !errors.Is(gerr, domain.ErrOrderNotFound) {
			return nil, false, gerr
		}
	}

	cart, err := s.cart.Get(ctx, rawToken, tenantID)
	if err != nil {
		return nil, false, err // cartclient.ErrEmptyCart / ErrUpstream -> httpx maps
	}

	rateID, err := uuid.Parse(req.ShippingRateID)
	if err != nil {
		return nil, false, scdomain.ErrShippingRateNotFound
	}
	couponCode := ""
	if req.CouponCode != nil {
		couponCode = *req.CouponCode
	}

	quote, err := s.storeconfig.QuoteFor(ctx, tenantID, cart.SubtotalCents, rateID, couponCode)
	if err != nil {
		return nil, false, err // store-config errors -> httpx maps
	}

	orderID := uuid.New()
	lines := stockLines(cart.Lines)
	if err := s.inventory.Reserve(ctx, tenantID, orderID.String(), lines); err != nil {
		return nil, false, err
	}

	billing := req.ShippingAddress
	if req.BillingAddress != nil {
		billing = *req.BillingAddress
	}

	order := &domain.Order{
		ID:              orderID,
		TenantID:        tenantID,
		CustomerID:      customerID,
		Status:          domain.StatusPendingPayment,
		Currency:        cart.Currency,
		SubtotalCents:   quote.SubtotalCents,
		DiscountCents:   quote.DiscountCents,
		ShippingCents:   quote.ShippingCents,
		TaxCents:        quote.TaxCents,
		GrandTotalCents: quote.GrandTotalCents,
		ItemCount:       itemCount(cart.Lines),
		CouponCode:      quote.CouponCode,
		ShippingAddress: req.ShippingAddress.JSON(),
		BillingAddress:  billing.JSON(),
	}
	if idempotencyKey != "" {
		order.IdempotencyKey = &idempotencyKey
	}
	items := orderItems(orderID, cart.Lines)

	// Redeeming the coupon (if any) runs inside the SAME transaction as the
	// order insert, so the two can't drift apart: a coupon that got exhausted
	// by a concurrent checkout between the quote above and now rolls this
	// order back entirely (no discount without a real redemption), and a
	// failed order insert never touches the coupon's counter.
	var couponID *uuid.UUID
	if quote.CouponID != nil {
		couponID = quote.CouponID
	}
	redeem := func(tx *sql.Tx) error {
		if couponID == nil {
			return nil
		}
		return storeconfigrepo.New(tx).Redeem(ctx, *couponID)
	}

	if err := s.repo.Create(ctx, order, items, redeem); err != nil {
		_ = s.inventory.Release(ctx, tenantID, orderID.String(), lines)
		if platform.IsUniqueViolation(err, "orders_idempotency_key") {
			existing, gerr := s.repo.GetByIdempotencyKey(ctx, tenantID, customerID, idempotencyKey)
			if gerr != nil {
				return nil, false, gerr
			}
			d, derr := s.detail(ctx, tenantID, existing.ID)
			return d, false, derr
		}
		return nil, false, err // e.g. storeconfig.ErrCouponExhausted — the whole order rolled back
	}

	_ = s.cart.Clear(ctx, rawToken, tenantID) // best effort

	s.publishOrder(ctx, order, events.OrderPlaced)

	// reload items so the response carries their real ids (the multi-row insert
	// above does not return them)
	if stored, lerr := s.repo.ListItems(ctx, order.ID); lerr == nil {
		items = stored
	}
	return &domain.Detail{Order: *order, Items: items}, true, nil
}

// ---------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------

func (s *Service) Get(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID) (*domain.Detail, error) {
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
			return nil, err
		}
	}
	items, err := s.repo.ListItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Detail{Order: *order, Items: items}, nil
}

// ListQuery is what the handler passes to List.
type ListQuery struct {
	All    bool // manager view: every customer's orders
	Status *domain.Status
	Page   platform.Page
}

func (s *Service) List(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, q ListQuery) ([]domain.Detail, error) {
	filter := domain.ListFilter{TenantID: tenantID, Status: q.Status, Page: q.Page}
	if q.All {
		if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
			return nil, err
		}
	} else {
		filter.CustomerID = &customerID
	}

	orders, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return []domain.Detail{}, nil
	}

	ids := make([]uuid.UUID, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
	}
	itemsByOrder, err := s.repo.ItemsByOrderIDs(ctx, ids) // one query, no N+1
	if err != nil {
		return nil, err
	}

	out := make([]domain.Detail, len(orders))
	for i := range orders {
		out[i] = domain.Detail{Order: orders[i], Items: itemsByOrder[orders[i].ID]}
	}
	return out, nil
}

// ---------------------------------------------------------------------
// Transitions
// ---------------------------------------------------------------------

// Pay creates (or re-syncs) the payment for an order via payment-service.
//   - COD: the payment is recorded PENDING and the order is CONFIRMED right away;
//     the cash is collected when the order is fulfilled.
//   - CARD: payment-service opens a Stripe PaymentIntent. If it captured
//     immediately (stub mode) the order is CONFIRMED + PAID; otherwise the
//     client_secret is returned for the browser to finish, and calling Pay again
//     re-checks Stripe.
func (s *Service) Pay(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID, req dto.PayRequest) (*PayResult, error) {
	order, err := s.load(ctx, tenantID, customerID, rawToken, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == domain.StatusFulfilled || order.Status == domain.StatusCancelled {
		return nil, domain.ErrInvalidTransition
	}

	method := domain.MethodCard
	if req.PaymentMethod != nil && *req.PaymentMethod != "" {
		method = domain.Method(strings.ToUpper(*req.PaymentMethod))
	}
	if !method.Valid() {
		return nil, domain.ErrInvalidPaymentMethod
	}

	var payment *paymentclient.Payment
	if order.PaymentID != nil {
		payment, err = s.payments.Sync(ctx, tenantID, *order.PaymentID)
	} else {
		payment, err = s.payments.Create(ctx, tenantID, paymentclient.CreateRequest{
			OrderID:     order.ID,
			CustomerID:  order.CustomerID,
			AmountCents: order.GrandTotalCents,
			Currency:    order.Currency,
			Method:      string(method),
		})
		if err == nil {
			if attachErr := s.repo.AttachPayment(ctx, tenantID, order.ID, payment.ID, domain.Method(payment.Method)); attachErr != nil {
				return nil, attachErr
			}
		}
	}
	if err != nil {
		return nil, err // paymentclient.ErrRejected / ErrUpstream -> httpx maps
	}

	if payment.Method == string(domain.MethodCOD) || payment.Status == "CAPTURED" {
		if err := s.advanceToConfirmed(ctx, tenantID, order); err != nil {
			return nil, err
		}
	}
	if payment.Status == "CAPTURED" {
		_ = s.repo.MarkPaymentPaid(ctx, tenantID, order.ID)
	}
	if payment.Status == "FAILED" {
		// Release whatever Checkout reserved so a declined card doesn't hold
		// stock forever. MarkPaymentFailed is guarded (only fires once, from
		// PENDING), so a repeated Pay/Sync call here is a safe no-op.
		if err := s.releaseOnPaymentFailure(ctx, tenantID, order.ID); err != nil {
			log.Printf("[order] release reservation for failed payment on order %s: %v", order.ID, err)
		}
	}

	detail, err := s.detail(ctx, tenantID, order.ID)
	if err != nil {
		return nil, err
	}
	if detail.Status == domain.StatusConfirmed {
		s.publishOrder(ctx, &detail.Order, events.OrderConfirmed)
	}
	return &PayResult{Detail: detail, ClientSecret: payment.ClientSecret}, nil
}

// advanceToConfirmed moves a PENDING_PAYMENT order to CONFIRMED, tolerating an
// order that is already past that point (idempotent Pay calls).
func (s *Service) advanceToConfirmed(ctx context.Context, tenantID uuid.UUID, order *domain.Order) error {
	if order.Status != domain.StatusPendingPayment {
		return nil
	}
	return s.repo.MarkConfirmed(ctx, tenantID, order.ID)
}

// releaseOnPaymentFailure marks the order's payment FAILED — a no-op once
// payment_status has moved off PENDING, so repeat delivery can't reprocess it
// — and, only when the order is still PENDING_PAYMENT, releases the stock
// Checkout reserved for it. If the order was already CANCELLED, Cancel()
// already released that stock; releasing again here would double-release the
// same units back into availability, so this is skipped for any order not
// still in PENDING_PAYMENT.
func (s *Service) releaseOnPaymentFailure(ctx context.Context, tenantID, orderID uuid.UUID) error {
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			return nil
		}
		return err
	}
	if err := s.repo.MarkPaymentFailed(ctx, tenantID, orderID); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			return nil // already handled (paid/refunded/already-failed) — nothing to release again
		}
		return err
	}
	if order.Status != domain.StatusPendingPayment {
		return nil // e.g. already CANCELLED — its stock was released by Cancel()
	}
	items, err := s.repo.ListItems(ctx, orderID)
	if err != nil {
		return err
	}
	return s.inventory.Release(ctx, tenantID, orderID.String(), stockLinesFromItems(items))
}

// ---------------------------------------------------------------------
// Kafka-driven sync — keeps the order in step with a webhook-driven payment
// capture/failure even if the customer never returns to call Pay again.
// ---------------------------------------------------------------------

// ApplyPaymentCaptured confirms and marks an order paid from a payment.captured
// event. Idempotent: a repeat delivery (or one that races an in-browser Pay
// call) finds the order already past PENDING_PAYMENT/PENDING and no-ops.
//
// If the order was CANCELLED before this capture was known about — a
// cancel-then-delayed-webhook race, which Cancel() defends against by voiding
// the PaymentIntent, but Stripe can still lose that race — the order is never
// resurrected to CONFIRMED. Instead the capture is recorded (money really did
// move) and immediately reversed through the existing refund path, exactly as
// if a manager had refunded a paid order.
func (s *Service) ApplyPaymentCaptured(ctx context.Context, tenantID, orderID uuid.UUID) error {
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return nil // stale/foreign event — nothing to apply
	}
	if err != nil {
		return err
	}

	switch order.Status {
	case domain.StatusPendingPayment, domain.StatusConfirmed:
		wasConfirmed := order.Status == domain.StatusConfirmed
		if err := s.advanceToConfirmed(ctx, tenantID, order); err != nil {
			return err
		}
		if err := s.repo.MarkPaymentPaid(ctx, tenantID, orderID); err != nil && !errors.Is(err, domain.ErrInvalidTransition) {
			return err
		}
		if wasConfirmed {
			return nil // this event didn't change anything — don't re-publish
		}
		detail, err := s.detail(ctx, tenantID, orderID)
		if err != nil {
			return err
		}
		s.publishOrder(ctx, &detail.Order, events.OrderConfirmed)
		return nil

	case domain.StatusCancelled:
		if order.PaymentID == nil {
			return nil
		}
		_ = s.repo.MarkPaymentPaid(ctx, tenantID, orderID) // record the real capture (idempotent guard)
		if _, err := s.payments.Refund(ctx, tenantID, *order.PaymentID, nil); err != nil {
			return err // retried by the consumer
		}
		_ = s.repo.MarkPaymentRefunded(ctx, tenantID, orderID)
		return nil

	default: // FULFILLED — already fully settled, nothing left to change
		return nil
	}
}

// ApplyPaymentFailed releases the order's reserved stock from a payment.failed
// event. Idempotent via releaseOnPaymentFailure's own guards.
func (s *Service) ApplyPaymentFailed(ctx context.Context, tenantID, orderID uuid.UUID) error {
	return s.releaseOnPaymentFailure(ctx, tenantID, orderID)
}

// Fulfil marks a CONFIRMED order FULFILLED and ships the reserved stock, storing
// the tracking info. For a COD order it also settles the payment. Managers only.
func (s *Service) Fulfil(ctx context.Context, tenantID uuid.UUID, rawToken string, orderID uuid.UUID, req dto.FulfilRequest) (*domain.Detail, error) {
	if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
		return nil, err
	}
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarkFulfilled(ctx, tenantID, order.ID, trim(req.Carrier), trim(req.TrackingNumber)); err != nil {
		return nil, err
	}
	if err := s.inventory.Ship(ctx, tenantID, order.ID.String(), stockLinesFromItems(items)); err != nil {
		return nil, err
	}

	if order.PaymentID != nil && order.PaymentStatus == string(domain.PaymentPending) {
		if _, err := s.payments.Settle(ctx, tenantID, *order.PaymentID); err == nil {
			_ = s.repo.MarkPaymentPaid(ctx, tenantID, order.ID)
		}
	}

	detail, err := s.detail(ctx, tenantID, order.ID)
	if err != nil {
		return nil, err
	}
	s.publishOrder(ctx, &detail.Order, events.OrderShipped)
	return detail, nil
}

// Cancel marks a PENDING_PAYMENT or CONFIRMED order CANCELLED, releases its stock
// and refunds a captured payment.
func (s *Service) Cancel(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID) (*domain.Detail, error) {
	order, err := s.load(ctx, tenantID, customerID, rawToken, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarkCancelled(ctx, tenantID, order.ID); err != nil {
		return nil, err
	}
	if err := s.inventory.Release(ctx, tenantID, order.ID.String(), stockLinesFromItems(items)); err != nil {
		return nil, err
	}

	refunded := false
	switch {
	case order.PaymentID != nil && order.PaymentStatus == string(domain.PaymentPaid):
		// Already captured — refund it (the existing refund path).
		if _, err := s.payments.Refund(ctx, tenantID, *order.PaymentID, nil); err == nil {
			_ = s.repo.MarkPaymentRefunded(ctx, tenantID, order.ID)
			refunded = true
		}
	case order.PaymentID != nil && order.PaymentStatus == string(domain.PaymentPending) &&
		order.PaymentMethod != nil && *order.PaymentMethod == string(domain.MethodCard):
		// Not captured yet — void the Stripe PaymentIntent so a delayed
		// capture can't land after the customer already cancelled. Best
		// effort: if Stripe rejects the void (it already captured a moment
		// earlier), the payment stays PENDING and ApplyPaymentCaptured will
		// detect the order is CANCELLED when that webhook arrives and refund
		// it automatically instead of resurrecting the order.
		if _, err := s.payments.Cancel(ctx, tenantID, *order.PaymentID); err != nil {
			log.Printf("[order] void pending payment for cancelled order %s: %v", order.ID, err)
		}
	}

	detail, err := s.detail(ctx, tenantID, order.ID)
	if err != nil {
		return nil, err
	}
	s.events.Publish(ctx, events.TopicOrders, detail.ID.String(), events.OrderCancelled,
		orderEventData(&detail.Order, map[string]any{"refunded": refunded}))
	return detail, nil
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

// load fetches an order the caller is allowed to act on (owner or manager).
func (s *Service) load(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (s *Service) detail(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.Detail, error) {
	order, err := s.repo.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &domain.Detail{Order: *order, Items: items}, nil
}

func (s *Service) requireManager(ctx context.Context, rawToken string, tenantID uuid.UUID) error {
	role, err := s.authz.RoleInTenant(ctx, rawToken, tenantID)
	if err != nil {
		return err // authz.ErrUpstream / ErrUnauthorized -> httpx maps
	}
	if role == authz.RoleAdmin || role == authz.RoleManager || role == authz.RoleSuperAdmin {
		return nil
	}
	return domain.ErrManagerOnly
}

// trim normalises an optional string, returning nil for blank.
func trim(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func itemCount(lines []cartclient.Line) int {
	n := 0
	for _, l := range lines {
		n += l.Quantity
	}
	return n
}

func stockLines(lines []cartclient.Line) []inventoryclient.Line {
	out := make([]inventoryclient.Line, len(lines))
	for i, l := range lines {
		out[i] = inventoryclient.Line{SKU: l.SKU, Quantity: l.Quantity}
	}
	return out
}

func stockLinesFromItems(items []domain.Item) []inventoryclient.Line {
	out := make([]inventoryclient.Line, len(items))
	for i, it := range items {
		out[i] = inventoryclient.Line{SKU: it.SKU, Quantity: it.Quantity}
	}
	return out
}

func orderItems(orderID uuid.UUID, lines []cartclient.Line) []domain.Item {
	out := make([]domain.Item, len(lines))
	for i, l := range lines {
		out[i] = domain.Item{
			OrderID:        orderID,
			ProductID:      l.ProductID,
			SKU:            l.SKU,
			Quantity:       l.Quantity,
			UnitPriceCents: l.UnitPriceCents,
			Currency:       l.Currency,
			ProductName:    l.ProductName,
			VariantTitle:   l.VariantTitle,
		}
	}
	return out
}
