package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

type CategoryLinkRepository struct {
	db platform.DBTX
}

func NewCategoryLinkRepository(db platform.DBTX) *CategoryLinkRepository {
	return &CategoryLinkRepository{db: db}
}

var _ domain.CategoryLinkRepository = (*CategoryLinkRepository)(nil)

// SetForProduct replaces the product's category links with exactly categoryIDs.
func (r *CategoryLinkRepository) SetForProduct(ctx context.Context, productID uuid.UUID, categoryIDs []uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM product_categories WHERE product_id = $1`, productID); err != nil {
		return err
	}
	if len(categoryIDs) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO product_categories (product_id, category_id)
		SELECT $1, unnest($2::uuid[])
		ON CONFLICT DO NOTHING`,
		productID, pq.Array(categoryIDs),
	)
	return err
}

func (r *CategoryLinkRepository) ListCategoryIDs(ctx context.Context, productID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT category_id FROM product_categories WHERE product_id = $1 ORDER BY category_id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
