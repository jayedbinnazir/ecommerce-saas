package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/platform"
	"github.com/jayedbinnazir/inventory-service/internal/stock/domain"
)

type MovementRepository struct {
	db platform.DBTX
}

func NewMovementRepository(db platform.DBTX) *MovementRepository {
	return &MovementRepository{db: db}
}

var _ domain.MovementRepository = (*MovementRepository)(nil)

const movementColumns = `
	id, stock_item_id, tenant_id, sku, type, quantity,
	on_hand_after, reserved_after, reason, reference, created_by, created_at`

func scanMovement(row platform.Scanner) (*domain.Movement, error) {
	var m domain.Movement
	if err := row.Scan(
		&m.ID, &m.StockItemID, &m.TenantID, &m.SKU, &m.Type, &m.Quantity,
		&m.OnHandAfter, &m.ReservedAfter, &m.Reason, &m.Reference, &m.CreatedBy, &m.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MovementRepository) Create(ctx context.Context, m *domain.Movement) error {
	const q = `
		INSERT INTO stock_movements (
			stock_item_id, tenant_id, sku, type, quantity,
			on_hand_after, reserved_after, reason, reference, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING ` + movementColumns

	created, err := scanMovement(r.db.QueryRowContext(ctx, q,
		m.StockItemID, m.TenantID, m.SKU, m.Type, m.Quantity,
		m.OnHandAfter, m.ReservedAfter, m.Reason, m.Reference, m.CreatedBy,
	))
	if err != nil {
		return err
	}
	*m = *created
	return nil
}

func (r *MovementRepository) ListByItem(ctx context.Context, stockItemID uuid.UUID, page platform.Page) ([]domain.Movement, error) {
	const q = `
		SELECT ` + movementColumns + `
		FROM stock_movements
		WHERE stock_item_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, stockItemID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Movement, 0)
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
