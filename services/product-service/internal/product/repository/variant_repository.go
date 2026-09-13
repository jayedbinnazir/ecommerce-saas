package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

type VariantRepository struct {
	db platform.DBTX
}

func NewVariantRepository(db platform.DBTX) *VariantRepository {
	return &VariantRepository{db: db}
}

var _ domain.VariantRepository = (*VariantRepository)(nil)

const variantColumns = `
	id, product_id, tenant_id, sku, title, price_cents, currency,
	compare_at_price_cents, weight_grams, barcode, position, is_default, created_at, updated_at`

func scanVariant(row platform.Scanner) (*domain.Variant, error) {
	var v domain.Variant
	err := row.Scan(
		&v.ID, &v.ProductID, &v.TenantID, &v.SKU, &v.Title, &v.PriceCents, &v.Currency,
		&v.CompareAtPriceCents, &v.WeightGrams, &v.Barcode, &v.Position, &v.IsDefault, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VariantRepository) Create(ctx context.Context, v *domain.Variant) error {
	const q = `
		INSERT INTO product_variants (
			product_id, tenant_id, sku, title, price_cents, currency,
			compare_at_price_cents, weight_grams, barcode, position, is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + variantColumns

	created, err := scanVariant(r.db.QueryRowContext(ctx, q,
		v.ProductID, v.TenantID, v.SKU, v.Title, v.PriceCents, v.Currency,
		v.CompareAtPriceCents, v.WeightGrams, v.Barcode, v.Position, v.IsDefault,
	))
	if err != nil {
		if platform.IsUniqueViolation(err, "product_variants_tenant_sku_key") {
			return domain.ErrSkuTaken
		}
		return err
	}
	*v = *created
	return nil
}

func (r *VariantRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Variant, error) {
	const q = `SELECT ` + variantColumns + ` FROM product_variants WHERE tenant_id = $1 AND id = $2`
	v, err := scanVariant(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrVariantNotFound
	}
	return v, err
}

func (r *VariantRepository) ListByProduct(ctx context.Context, productID uuid.UUID) ([]domain.Variant, error) {
	const q = `SELECT ` + variantColumns + ` FROM product_variants WHERE product_id = $1 ORDER BY position, created_at`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Variant, 0)
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *VariantRepository) CountByProduct(ctx context.Context, productID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM product_variants WHERE product_id = $1`, productID).Scan(&n)
	return n, err
}

func (r *VariantRepository) Update(ctx context.Context, v *domain.Variant) error {
	const q = `
		UPDATE product_variants SET
			sku = $3, title = $4, price_cents = $5, currency = $6,
			compare_at_price_cents = $7, weight_grams = $8, barcode = $9,
			position = $10, is_default = $11
		WHERE id = $1 AND tenant_id = $2
		RETURNING ` + variantColumns

	updated, err := scanVariant(r.db.QueryRowContext(ctx, q,
		v.ID, v.TenantID, v.SKU, v.Title, v.PriceCents, v.Currency,
		v.CompareAtPriceCents, v.WeightGrams, v.Barcode, v.Position, v.IsDefault,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrVariantNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "product_variants_tenant_sku_key") {
			return domain.ErrSkuTaken
		}
		return err
	}
	*v = *updated
	return nil
}

func (r *VariantRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM product_variants WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrVariantNotFound)
}

func (r *VariantRepository) ClearDefault(ctx context.Context, tenantID, productID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE product_variants SET is_default = false WHERE tenant_id = $1 AND product_id = $2 AND is_default`,
		tenantID, productID)
	return err
}
