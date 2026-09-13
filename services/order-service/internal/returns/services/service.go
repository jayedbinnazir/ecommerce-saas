// Package services holds the returns-module logic: a customer requests a return
// on a fulfilled order; a manager approves (refund + optional restock) or rejects.
package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/inventoryclient"
	orderdomain "github.com/jayedbinnazir/order-service/internal/order/domain"
	orderrepo "github.com/jayedbinnazir/order-service/internal/order/repository"
	"github.com/jayedbinnazir/order-service/internal/paymentclient"
	"github.com/jayedbinnazir/order-service/internal/platform"
	"github.com/jayedbinnazir/order-service/internal/returns/domain"
	"github.com/jayedbinnazir/order-service/internal/returns/dto"
)

type Service struct {
	repo      domain.Repository
	orders    *orderrepo.Repository
	payments  *paymentclient.Client
	inventory *inventoryclient.Client
	authz     *authz.Client
}

func New(repo domain.Repository, orders *orderrepo.Repository, payments *paymentclient.Client, inventory *inventoryclient.Client, authzClient *authz.Client) *Service {
	return &Service{repo: repo, orders: orders, payments: payments, inventory: inventory, authz: authzClient}
}

// Request records a return for a fulfilled order the caller owns.
func (s *Service) Request(ctx context.Context, tenantID, customerID, orderID uuid.UUID, req dto.RequestReturnRequest) (*domain.Detail, error) {
	order, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		return nil, domain.ErrNotOwner
	}
	if order.Status != orderdomain.StatusFulfilled {
		return nil, domain.ErrOrderNotReturnable
	}

	orderItems, err := s.orders.ListItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	bySKU := make(map[string]orderdomain.Item, len(orderItems))
	for _, it := range orderItems {
		bySKU[strings.ToLower(it.SKU)] = it
	}

	// One query for every SKU already covered by a pending or completed return
	// on this order, so a second request can't push the total returned past
	// what was actually ordered (a rejected return doesn't count — rejecting
	// one frees its quantity back up).
	alreadyReturned, err := s.repo.AlreadyReturnedBySKU(ctx, orderID)
	if err != nil {
		return nil, err
	}

	items := make([]domain.Item, 0, len(req.Items))
	var suggested int64
	for _, in := range req.Items {
		sku := strings.ToLower(strings.TrimSpace(in.SKU))
		oi, ok := bySKU[sku]
		if !ok || in.Quantity > oi.Quantity {
			return nil, domain.ErrInvalidReturnItems
		}
		if alreadyReturned[sku]+in.Quantity > oi.Quantity {
			return nil, domain.ErrReturnQuantityExceeded
		}
		items = append(items, domain.Item{
			SKU:            oi.SKU,
			Quantity:       in.Quantity,
			UnitPriceCents: oi.UnitPriceCents,
			ProductName:    oi.ProductName,
		})
		suggested += int64(in.Quantity) * oi.UnitPriceCents
	}
	if len(items) == 0 {
		return nil, domain.ErrNoReturnItems
	}

	ret := &domain.Return{
		OrderID:     orderID,
		TenantID:    tenantID,
		CustomerID:  customerID,
		Status:      domain.StatusRequested,
		Reason:      strings.TrimSpace(req.Reason),
		RefundCents: suggested,
	}
	if err := s.repo.Create(ctx, ret, items); err != nil {
		return nil, err
	}
	return &domain.Detail{Return: *ret, Items: items}, nil
}

// Get returns a return the caller may see (owner or manager).
func (s *Service) Get(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, returnID uuid.UUID) (*domain.Detail, error) {
	ret, err := s.repo.GetByID(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}
	if ret.CustomerID != customerID {
		if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
			return nil, err
		}
	}
	return s.detail(ctx, ret)
}

// ListForOrder returns the returns raised against one order (owner or manager).
func (s *Service) ListForOrder(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID) ([]domain.Detail, error) {
	order, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
			return nil, err
		}
	}
	rets, err := s.repo.ListByOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	return s.stitch(ctx, rets)
}

// List is the tenant-wide manager view.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, rawToken string, status *domain.Status, page platform.Page) ([]domain.Detail, error) {
	if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
		return nil, err
	}
	rets, err := s.repo.List(ctx, domain.ListFilter{TenantID: tenantID, Status: status, Page: page})
	if err != nil {
		return nil, err
	}
	return s.stitch(ctx, rets)
}

// Resolve approves (refund + optional restock) or rejects a return. Managers only.
func (s *Service) Resolve(ctx context.Context, tenantID uuid.UUID, rawToken string, returnID uuid.UUID, req dto.ResolveReturnRequest) (*domain.Detail, error) {
	if err := s.requireManager(ctx, rawToken, tenantID); err != nil {
		return nil, err
	}
	ret, err := s.repo.GetByID(ctx, tenantID, returnID)
	if err != nil {
		return nil, err
	}
	if ret.Status != domain.StatusRequested {
		return nil, domain.ErrAlreadyResolved
	}
	items, err := s.repo.ListItems(ctx, returnID)
	if err != nil {
		return nil, err
	}

	if !req.Approve {
		if err := s.repo.Resolve(ctx, tenantID, returnID, domain.StatusRejected, trim(req.Note), 0, false); err != nil {
			return nil, err
		}
		return s.reload(ctx, tenantID, returnID)
	}

	refundCents := ret.RefundCents
	if req.RefundCents != nil {
		refundCents = *req.RefundCents
	}

	order, err := s.orders.GetByID(ctx, tenantID, ret.OrderID)
	if err != nil {
		return nil, err
	}
	if order.PaymentID != nil && refundCents > 0 {
		if _, err := s.payments.Refund(ctx, tenantID, *order.PaymentID, &refundCents); err != nil {
			return nil, err // paymentclient.ErrRejected / ErrUpstream -> httpx maps
		}
		_ = s.orders.MarkPaymentRefunded(ctx, tenantID, order.ID) // best-effort mirror
	}

	restocked := false
	if req.Restock == nil || *req.Restock {
		lines := make([]inventoryclient.Line, len(items))
		for i, it := range items {
			lines[i] = inventoryclient.Line{SKU: it.SKU, Quantity: it.Quantity}
		}
		if err := s.inventory.Restock(ctx, tenantID, returnID.String(), lines); err == nil {
			restocked = true
		}
	}

	if err := s.repo.Resolve(ctx, tenantID, returnID, domain.StatusCompleted, trim(req.Note), refundCents, restocked); err != nil {
		return nil, err
	}
	return s.reload(ctx, tenantID, returnID)
}

// ---- helpers ----

func (s *Service) reload(ctx context.Context, tenantID, id uuid.UUID) (*domain.Detail, error) {
	ret, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, ret)
}

func (s *Service) detail(ctx context.Context, ret *domain.Return) (*domain.Detail, error) {
	items, err := s.repo.ListItems(ctx, ret.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Detail{Return: *ret, Items: items}, nil
}

// stitch attaches items to a page of returns in ONE query (no N+1).
func (s *Service) stitch(ctx context.Context, rets []domain.Return) ([]domain.Detail, error) {
	if len(rets) == 0 {
		return []domain.Detail{}, nil
	}
	ids := make([]uuid.UUID, len(rets))
	for i := range rets {
		ids[i] = rets[i].ID
	}
	byRet, err := s.repo.ItemsByReturnIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Detail, len(rets))
	for i := range rets {
		out[i] = domain.Detail{Return: rets[i], Items: byRet[rets[i].ID]}
	}
	return out, nil
}

func (s *Service) requireManager(ctx context.Context, rawToken string, tenantID uuid.UUID) error {
	role, err := s.authz.RoleInTenant(ctx, rawToken, tenantID)
	if err != nil {
		return err
	}
	switch role {
	case authz.RoleAdmin, authz.RoleManager, authz.RoleSuperAdmin:
		return nil
	default:
		return domain.ErrManagerOnly
	}
}

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
