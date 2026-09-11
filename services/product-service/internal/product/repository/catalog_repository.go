package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

// ---------------------------------------------------------------------
// Product display specs  (product_specs)
// ---------------------------------------------------------------------

type SpecRepository struct {
	db platform.DBTX
}

func NewSpecRepository(db platform.DBTX) *SpecRepository { return &SpecRepository{db: db} }

func (r *SpecRepository) ListByProduct(ctx context.Context, productID uuid.UUID) ([]domain.ProductSpec, error) {
	const q = `SELECT product_id, attribute_id, value FROM product_specs WHERE product_id = $1`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ProductSpec, 0)
	for rows.Next() {
		var s domain.ProductSpec
		if err := rows.Scan(&s.ProductID, &s.AttributeID, &s.Value); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Replace swaps a product's whole spec set: one DELETE, then one multi-row INSERT.
func (r *SpecRepository) Replace(ctx context.Context, productID uuid.UUID, specs []domain.ProductSpec) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM product_specs WHERE product_id = $1`, productID); err != nil {
		return err
	}
	if len(specs) == 0 {
		return nil
	}

	values := make([]string, 0, len(specs))
	args := make([]any, 0, len(specs)*3)
	for i, s := range specs {
		base := i * 3
		values = append(values, "($"+itoa(base+1)+", $"+itoa(base+2)+", $"+itoa(base+3)+")")
		args = append(args, productID, s.AttributeID, s.Value)
	}
	q := `INSERT INTO product_specs (product_id, attribute_id, value) VALUES ` + strings.Join(values, ", ")
	_, err := r.db.ExecContext(ctx, q, args...)
	if platform.IsForeignKeyViolation(err) {
		return domain.ErrAttributeNotForProduct
	}
	return err
}

// ---------------------------------------------------------------------
// Variant option selections  (product_variant_options)
// ---------------------------------------------------------------------

type VariantOptionRepository struct {
	db platform.DBTX
}

func NewVariantOptionRepository(db platform.DBTX) *VariantOptionRepository {
	return &VariantOptionRepository{db: db}
}

// SetForVariant replaces a variant's option rows: one DELETE, then one multi-row
// INSERT. The (attribute_id, attribute_value_id) pair is checked by a composite
// foreign key, so a mismatched value fails here.
func (r *VariantOptionRepository) SetForVariant(ctx context.Context, variantID uuid.UUID, opts []domain.VariantOption) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM product_variant_options WHERE variant_id = $1`, variantID); err != nil {
		return err
	}
	if len(opts) == 0 {
		return nil
	}

	values := make([]string, 0, len(opts))
	args := make([]any, 0, len(opts)*3)
	for i, o := range opts {
		base := i * 3
		values = append(values, "($"+itoa(base+1)+", $"+itoa(base+2)+", $"+itoa(base+3)+")")
		args = append(args, variantID, o.AttributeID, o.AttributeValueID)
	}
	q := `INSERT INTO product_variant_options (variant_id, attribute_id, attribute_value_id) VALUES ` +
		strings.Join(values, ", ")
	_, err := r.db.ExecContext(ctx, q, args...)
	if platform.IsForeignKeyViolation(err) {
		return domain.ErrOptionValueInvalid
	}
	return err
}

// ByVariants returns the option rows for many variants in one query (no N+1).
func (r *VariantOptionRepository) ByVariants(ctx context.Context, variantIDs []uuid.UUID) ([]domain.VariantOption, error) {
	if len(variantIDs) == 0 {
		return nil, nil
	}
	ids := make([]string, len(variantIDs))
	for i, id := range variantIDs {
		ids[i] = id.String()
	}
	const q = `SELECT variant_id, attribute_id, attribute_value_id
		FROM product_variant_options WHERE variant_id = ANY($1)`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.VariantOption, 0)
	for rows.Next() {
		var o domain.VariantOption
		if err := rows.Scan(&o.VariantID, &o.AttributeID, &o.AttributeValueID); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
