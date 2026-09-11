// Package services holds the cart-module application logic: one active cart per
// shopper, with price/name snapshots taken from product-service and a stock
// check against inventory-service.
package services

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/cart-service/internal/availability"
	"github.com/jayedbinnazir/cart-service/internal/cart/domain"
	"github.com/jayedbinnazir/cart-service/internal/cart/dto"
	"github.com/jayedbinnazir/cart-service/internal/cart/repository"
	"github.com/jayedbinnazir/cart-service/internal/catalog"
	"github.com/jayedbinnazir/cart-service/internal/platform"
)

var skuRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

type Service struct {
	db        *sql.DB
	catalog   *catalog.Client
	inventory *availability.Client
}

func New(db *sql.DB, catalogClient *catalog.Client, inventoryClient *availability.Client) *Service {
	return &Service{db: db, catalog: catalogClient, inventory: inventoryClient}
}

// GetCart returns the shopper's active cart, creating an empty one if needed.
func (s *Service) GetCart(ctx context.Context, tenantID, customerID uuid.UUID) (*domain.Detail, error) {
	cart, err := s.getOrCreateActiveCart(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, cart)
}

// AddItem adds quantity of sku to the cart. If the line already exists its
// quantity is increased; the original price snapshot is kept.
func (s *Service) AddItem(ctx context.Context, tenantID, customerID uuid.UUID, rawToken string, req dto.AddItemRequest) (*domain.Detail, error) {
	sku := strings.TrimSpace(req.SKU)
	if !skuRE.MatchString(sku) {
		return nil, domain.ErrInvalidSKU
	}
	if req.Quantity < 1 {
		return nil, domain.ErrInvalidQuantity
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, domain.ErrVariantNotAvailable
	}

	variant, err := s.lookupVariant(ctx, rawToken, tenantID, productID, sku)
	if err != nil {
		return nil, err
	}

	cart, err := s.getOrCreateActiveCart(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	itemRepo := repository.NewItemRepository(s.db)
	existing, err := itemRepo.GetBySKU(ctx, cart.ID, sku)
	if err != nil && !errors.Is(err, domain.ErrItemNotFound) {
		return nil, err
	}

	wantQty := req.Quantity
	if existing != nil {
		if existing.Currency != variant.Currency {
			return nil, domain.ErrCurrencyMismatch
		}
		wantQty += existing.Quantity
	} else if err := s.checkCartCurrency(ctx, cart.ID, variant.Currency); err != nil {
		return nil, err
	}

	if err := s.checkStock(ctx, rawToken, tenantID, sku, wantQty); err != nil {
		return nil, err
	}

	if existing != nil {
		if err := itemRepo.SetQuantity(ctx, existing.ID, wantQty); err != nil {
			return nil, err
		}
	} else {
		item := &domain.Item{
			CartID:         cart.ID,
			ProductID:      variant.ProductID,
			SKU:            variant.SKU,
			Quantity:       wantQty,
			UnitPriceCents: variant.PriceCents,
			Currency:       variant.Currency,
			ProductName:    variant.ProductName,
			VariantTitle:   variant.VariantTitle,
		}
		if err := itemRepo.Create(ctx, item); err != nil {
			return nil, err
		}
	}

	_ = repository.NewCartRepository(s.db).Touch(ctx, cart.ID)
	return s.detail(ctx, cart)
}

// SetItemQuantity replaces the quantity of an existing line (use RemoveItem for 0).
func (s *Service) SetItemQuantity(ctx context.Context, tenantID, customerID uuid.UUID, rawToken, sku string, req dto.UpdateItemRequest) (*domain.Detail, error) {
	sku = strings.TrimSpace(sku)
	if req.Quantity < 1 {
		return nil, domain.ErrInvalidQuantity
	}

	cart, err := s.getOrCreateActiveCart(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}

	itemRepo := repository.NewItemRepository(s.db)
	item, err := itemRepo.GetBySKU(ctx, cart.ID, sku)
	if err != nil {
		return nil, err
	}

	if err := s.checkStock(ctx, rawToken, tenantID, sku, req.Quantity); err != nil {
		return nil, err
	}
	if err := itemRepo.SetQuantity(ctx, item.ID, req.Quantity); err != nil {
		return nil, err
	}

	_ = repository.NewCartRepository(s.db).Touch(ctx, cart.ID)
	return s.detail(ctx, cart)
}

// RemoveItem deletes a line from the cart.
func (s *Service) RemoveItem(ctx context.Context, tenantID, customerID uuid.UUID, sku string) (*domain.Detail, error) {
	cart, err := s.getOrCreateActiveCart(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if err := repository.NewItemRepository(s.db).Delete(ctx, cart.ID, strings.TrimSpace(sku)); err != nil {
		return nil, err
	}
	_ = repository.NewCartRepository(s.db).Touch(ctx, cart.ID)
	return s.detail(ctx, cart)
}

// Clear removes every line but keeps the (now empty) cart.
func (s *Service) Clear(ctx context.Context, tenantID, customerID uuid.UUID) (*domain.Detail, error) {
	cart, err := s.getOrCreateActiveCart(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	if err := repository.NewItemRepository(s.db).DeleteAllByCart(ctx, cart.ID); err != nil {
		return nil, err
	}
	_ = repository.NewCartRepository(s.db).Touch(ctx, cart.ID)
	return s.detail(ctx, cart)
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func (s *Service) getOrCreateActiveCart(ctx context.Context, tenantID, customerID uuid.UUID) (*domain.Cart, error) {
	repo := repository.NewCartRepository(s.db)

	cart, err := repo.GetActive(ctx, tenantID, customerID)
	if err == nil {
		return cart, nil
	}
	if !errors.Is(err, domain.ErrCartNotFound) {
		return nil, err
	}

	cart = &domain.Cart{TenantID: tenantID, CustomerID: customerID, Status: domain.StatusActive}
	err = repo.Create(ctx, cart)
	if err == nil {
		return cart, nil
	}
	// lost a race to create the one active cart — read the winner
	if platform.IsUniqueViolation(err, "carts_one_active_per_customer") {
		return repo.GetActive(ctx, tenantID, customerID)
	}
	return nil, err
}

func (s *Service) detail(ctx context.Context, cart *domain.Cart) (*domain.Detail, error) {
	items, err := repository.NewItemRepository(s.db).ListByCart(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	return &domain.Detail{Cart: *cart, Items: items}, nil
}

func (s *Service) lookupVariant(ctx context.Context, rawToken string, tenantID, productID uuid.UUID, sku string) (*catalog.Variant, error) {
	variant, err := s.catalog.Lookup(ctx, rawToken, tenantID, productID, sku)
	switch {
	case errors.Is(err, catalog.ErrNotAvailable):
		return nil, domain.ErrVariantNotAvailable
	case err != nil:
		return nil, err // catalog.ErrUpstream -> httpx maps to 502
	}
	return variant, nil
}

func (s *Service) checkCartCurrency(ctx context.Context, cartID uuid.UUID, currency string) error {
	items, err := repository.NewItemRepository(s.db).ListByCart(ctx, cartID)
	if err != nil {
		return err
	}
	if len(items) > 0 && items[0].Currency != currency {
		return domain.ErrCurrencyMismatch
	}
	return nil
}

func (s *Service) checkStock(ctx context.Context, rawToken string, tenantID uuid.UUID, sku string, want int) error {
	stock, err := s.inventory.Get(ctx, rawToken, tenantID, sku)
	if err != nil {
		return err // availability.ErrUpstream -> httpx maps to 502
	}
	if !stock.Enough(want) {
		return domain.ErrInsufficientStock
	}
	return nil
}
