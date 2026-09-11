// Package services holds the stock-module application logic: tracking on-hand and
// reserved quantities per SKU and writing an append-only movement ledger.
package services

import (
	"context"
	"database/sql"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/platform"
	"github.com/jayedbinnazir/inventory-service/internal/stock/domain"
	"github.com/jayedbinnazir/inventory-service/internal/stock/dto"
	"github.com/jayedbinnazir/inventory-service/internal/stock/repository"
)

var skuRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

// ---------------------------------------------------------------------
// Stock items (metadata: which SKUs are tracked, thresholds, location)
// ---------------------------------------------------------------------

func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req dto.CreateStockItemRequest) (*domain.StockItem, error) {
	sku := strings.TrimSpace(req.SKU)
	if !skuRE.MatchString(sku) {
		return nil, domain.ErrInvalidSKU
	}

	item := &domain.StockItem{
		TenantID:     tenantID,
		SKU:          sku,
		ReorderLevel: req.ReorderLevel,
		Location:     trimPtr(req.Location),
	}
	if err := repository.NewStockRepository(s.db).Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.StockItem, error) {
	return repository.NewStockRepository(s.db).GetBySKU(ctx, tenantID, cleanSKU(sku))
}

func (s *Service) List(ctx context.Context, f domain.ListFilter) ([]domain.StockItem, error) {
	return repository.NewStockRepository(s.db).List(ctx, f)
}

func (s *Service) Update(ctx context.Context, tenantID uuid.UUID, sku string, req dto.UpdateStockItemRequest) (*domain.StockItem, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}

	repo := repository.NewStockRepository(s.db)
	item, err := repo.GetBySKU(ctx, tenantID, cleanSKU(sku))
	if err != nil {
		return nil, err
	}

	if req.ReorderLevel != nil {
		item.ReorderLevel = *req.ReorderLevel
	}
	if req.Location != nil {
		item.Location = trimPtr(req.Location)
	}

	if err := repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) Delete(ctx context.Context, tenantID uuid.UUID, sku string) error {
	return repository.NewStockRepository(s.db).Delete(ctx, tenantID, cleanSKU(sku))
}

// ---------------------------------------------------------------------
// Movements (the operations that change quantities)
// ---------------------------------------------------------------------

// change is what one operation does to a locked stock item.
type change struct {
	movementType domain.MovementType
	quantity     int
	reason       *string
	reference    *string
	actor        *uuid.UUID
	// apply mutates item, or returns a domain error if the change is not allowed.
	apply func(item *domain.StockItem) error
}

func (s *Service) Receive(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.ReceiveRequest) (*domain.StockItem, error) {
	return s.mutate(ctx, tenantID, sku, change{
		movementType: domain.MovementReceive,
		quantity:     req.Quantity,
		reason:       trimPtr(req.Reason),
		reference:    trimPtr(req.Reference),
		actor:        actorPtr(actor),
		apply: func(item *domain.StockItem) error {
			item.OnHand += req.Quantity
			return nil
		},
	})
}

func (s *Service) Adjust(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.AdjustRequest) (*domain.StockItem, error) {
	if req.Quantity == 0 {
		return nil, domain.ErrInvalidQuantity
	}
	return s.mutate(ctx, tenantID, sku, change{
		movementType: domain.MovementAdjust,
		quantity:     req.Quantity,
		reason:       trimPtr(req.Reason),
		actor:        actorPtr(actor),
		apply: func(item *domain.StockItem) error {
			next := item.OnHand + req.Quantity
			if next < 0 || next < item.Reserved {
				return domain.ErrWouldGoNegative
			}
			item.OnHand = next
			return nil
		},
	})
}

func (s *Service) Reserve(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.MoveRequest) (*domain.StockItem, error) {
	return s.mutate(ctx, tenantID, sku, change{
		movementType: domain.MovementReserve,
		quantity:     req.Quantity,
		reference:    trimPtr(req.Reference),
		actor:        actorPtr(actor),
		apply: func(item *domain.StockItem) error {
			if item.Available() < req.Quantity {
				return domain.ErrInsufficientStock
			}
			item.Reserved += req.Quantity
			return nil
		},
	})
}

func (s *Service) Release(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.MoveRequest) (*domain.StockItem, error) {
	return s.mutate(ctx, tenantID, sku, change{
		movementType: domain.MovementRelease,
		quantity:     req.Quantity,
		reference:    trimPtr(req.Reference),
		actor:        actorPtr(actor),
		apply: func(item *domain.StockItem) error {
			if item.Reserved < req.Quantity {
				return domain.ErrInsufficientReserved
			}
			item.Reserved -= req.Quantity
			return nil
		},
	})
}

func (s *Service) Ship(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.MoveRequest) (*domain.StockItem, error) {
	return s.mutate(ctx, tenantID, sku, change{
		movementType: domain.MovementShip,
		quantity:     req.Quantity,
		reference:    trimPtr(req.Reference),
		actor:        actorPtr(actor),
		apply: func(item *domain.StockItem) error {
			if item.Reserved < req.Quantity {
				return domain.ErrInsufficientReserved
			}
			if item.OnHand < req.Quantity {
				return domain.ErrInsufficientStock
			}
			item.OnHand -= req.Quantity
			item.Reserved -= req.Quantity
			return nil
		},
	})
}

// ---------------------------------------------------------------------
// Bulk operations (one transaction, all-or-nothing) — used by order-service
// ---------------------------------------------------------------------

// applyLine mutates a locked item by qty, or returns a domain error.
type applyLine func(item *domain.StockItem, qty int) error

func (s *Service) ReserveBulk(ctx context.Context, tenantID, actor uuid.UUID, req dto.BulkMoveRequest) ([]domain.StockItem, error) {
	return s.mutateBulk(ctx, tenantID, actor, domain.MovementReserve, req, func(item *domain.StockItem, qty int) error {
		if item.Available() < qty {
			return domain.ErrInsufficientStock
		}
		item.Reserved += qty
		return nil
	})
}

func (s *Service) ReleaseBulk(ctx context.Context, tenantID, actor uuid.UUID, req dto.BulkMoveRequest) ([]domain.StockItem, error) {
	return s.mutateBulk(ctx, tenantID, actor, domain.MovementRelease, req, func(item *domain.StockItem, qty int) error {
		if item.Reserved < qty {
			return domain.ErrInsufficientReserved
		}
		item.Reserved -= qty
		return nil
	})
}

func (s *Service) ShipBulk(ctx context.Context, tenantID, actor uuid.UUID, req dto.BulkMoveRequest) ([]domain.StockItem, error) {
	return s.mutateBulk(ctx, tenantID, actor, domain.MovementShip, req, func(item *domain.StockItem, qty int) error {
		if item.Reserved < qty {
			return domain.ErrInsufficientReserved
		}
		if item.OnHand < qty {
			return domain.ErrInsufficientStock
		}
		item.OnHand -= qty
		item.Reserved -= qty
		return nil
	})
}

// RestockBulk puts returned units back on the shelf (on_hand += qty), recorded
// as a RECEIVE movement referencing the return.
func (s *Service) RestockBulk(ctx context.Context, tenantID, actor uuid.UUID, req dto.BulkMoveRequest) ([]domain.StockItem, error) {
	return s.mutateBulk(ctx, tenantID, actor, domain.MovementReceive, req, func(item *domain.StockItem, qty int) error {
		item.OnHand += qty
		return nil
	})
}

// mutateBulk locks every affected row (in sku order, to avoid deadlocks),
// applies the change and records one movement per line, in a single transaction.
func (s *Service) mutateBulk(ctx context.Context, tenantID, actor uuid.UUID, mvType domain.MovementType, req dto.BulkMoveRequest, apply applyLine) ([]domain.StockItem, error) {
	lines := mergeLines(req.Items)
	reference := trimPtr(req.Reference)
	who := actorPtr(actor)

	out := make([]domain.StockItem, 0, len(lines))
	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		stockRepo := repository.NewStockRepository(tx)
		moveRepo := repository.NewMovementRepository(tx)

		for _, ln := range lines {
			item, err := stockRepo.LockBySKU(ctx, tenantID, ln.sku)
			if err != nil {
				return err
			}
			if err := apply(item, ln.qty); err != nil {
				return err
			}
			if err := stockRepo.Update(ctx, item); err != nil {
				return err
			}
			if err := moveRepo.Create(ctx, &domain.Movement{
				StockItemID:   item.ID,
				TenantID:      tenantID,
				SKU:           item.SKU,
				Type:          mvType,
				Quantity:      ln.qty,
				OnHandAfter:   item.OnHand,
				ReservedAfter: item.Reserved,
				Reference:     reference,
				CreatedBy:     who,
			}); err != nil {
				return err
			}
			out = append(out, *item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type mergedLine struct {
	sku string
	qty int
}

// mergeLines folds duplicate SKUs together and sorts by SKU so every bulk call
// locks rows in the same order.
func mergeLines(items []dto.BulkLine) []mergedLine {
	bySKU := make(map[string]int, len(items))
	for _, it := range items {
		bySKU[cleanSKU(it.SKU)] += it.Quantity
	}
	out := make([]mergedLine, 0, len(bySKU))
	for sku, qty := range bySKU {
		out = append(out, mergedLine{sku: sku, qty: qty})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].sku < out[j].sku })
	return out
}

func (s *Service) Movements(ctx context.Context, tenantID uuid.UUID, sku string, page platform.Page) ([]domain.Movement, error) {
	repo := repository.NewStockRepository(s.db)
	item, err := repo.GetBySKU(ctx, tenantID, cleanSKU(sku))
	if err != nil {
		return nil, err
	}
	return repository.NewMovementRepository(s.db).ListByItem(ctx, item.ID, page)
}

// mutate locks the stock item, applies the change, saves it and records one
// movement — all in a single transaction.
func (s *Service) mutate(ctx context.Context, tenantID uuid.UUID, sku string, ch change) (*domain.StockItem, error) {
	sku = cleanSKU(sku)

	var result *domain.StockItem
	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		stockRepo := repository.NewStockRepository(tx)

		item, err := stockRepo.LockBySKU(ctx, tenantID, sku)
		if err != nil {
			return err
		}
		if err := ch.apply(item); err != nil {
			return err
		}
		if err := stockRepo.Update(ctx, item); err != nil {
			return err
		}

		movement := &domain.Movement{
			StockItemID:   item.ID,
			TenantID:      tenantID,
			SKU:           item.SKU,
			Type:          ch.movementType,
			Quantity:      ch.quantity,
			OnHandAfter:   item.OnHand,
			ReservedAfter: item.Reserved,
			Reason:        ch.reason,
			Reference:     ch.reference,
			CreatedBy:     ch.actor,
		}
		if err := repository.NewMovementRepository(tx).Create(ctx, movement); err != nil {
			return err
		}

		result = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func cleanSKU(s string) string { return strings.TrimSpace(s) }

func actorPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
