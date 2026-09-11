// Package repository is the database/sql implementation of the attribute repo.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/product-service/internal/attributes/domain"
	"github.com/jayedbinnazir/product-service/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const attrColumns = `id, tenant_id, category_id, name, code, role, position, created_at, updated_at`

func scanAttr(row platform.Scanner) (*domain.Attribute, error) {
	var a domain.Attribute
	if err := row.Scan(&a.ID, &a.TenantID, &a.CategoryID, &a.Name, &a.Code, &a.Role, &a.Position, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

const valueColumns = `id, attribute_id, value, position, created_at`

func scanValue(row platform.Scanner) (*domain.Value, error) {
	var v domain.Value
	if err := row.Scan(&v.ID, &v.AttributeID, &v.Value, &v.Position, &v.CreatedAt); err != nil {
		return nil, err
	}
	return &v, nil
}

// ---- attributes ----

func (r *Repository) CreateAttribute(ctx context.Context, a *domain.Attribute) error {
	const q = `
		INSERT INTO product_attributes (tenant_id, category_id, name, code, role, position)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + attrColumns

	created, err := scanAttr(r.db.QueryRowContext(ctx, q, a.TenantID, a.CategoryID, a.Name, a.Code, a.Role, a.Position))
	if err != nil {
		if platform.IsUniqueViolation(err, "product_attributes_category_code_key") {
			return domain.ErrCodeTaken
		}
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrCategoryNotFound
		}
		return err
	}
	*a = *created
	return nil
}

func (r *Repository) GetAttribute(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attribute, error) {
	const q = `SELECT ` + attrColumns + ` FROM product_attributes WHERE tenant_id = $1 AND id = $2`
	a, err := scanAttr(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAttributeNotFound
	}
	return a, err
}

func (r *Repository) ListByCategory(ctx context.Context, tenantID, categoryID uuid.UUID) ([]domain.Attribute, error) {
	const q = `SELECT ` + attrColumns + `
		FROM product_attributes
		WHERE tenant_id = $1 AND category_id = $2
		ORDER BY position, name`
	return r.queryAttrs(ctx, q, tenantID, categoryID)
}

func (r *Repository) ListByCategories(ctx context.Context, categoryIDs []uuid.UUID) ([]domain.Attribute, error) {
	if len(categoryIDs) == 0 {
		return nil, nil
	}
	const q = `SELECT ` + attrColumns + `
		FROM product_attributes
		WHERE category_id = ANY($1)
		ORDER BY position, name`
	return r.queryAttrs(ctx, q, pq.Array(uuidStrings(categoryIDs)))
}

func (r *Repository) queryAttrs(ctx context.Context, q string, args ...any) ([]domain.Attribute, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Attribute, 0)
	for rows.Next() {
		a, err := scanAttr(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateAttribute(ctx context.Context, a *domain.Attribute) error {
	const q = `
		UPDATE product_attributes SET name = $3, code = $4, role = $5, position = $6
		WHERE tenant_id = $1 AND id = $2
		RETURNING ` + attrColumns

	updated, err := scanAttr(r.db.QueryRowContext(ctx, q, a.TenantID, a.ID, a.Name, a.Code, a.Role, a.Position))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrAttributeNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "product_attributes_category_code_key") {
			return domain.ErrCodeTaken
		}
		return err
	}
	*a = *updated
	return nil
}

func (r *Repository) DeleteAttribute(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM product_attributes WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrAttributeNotFound)
}

// ---- values ----

func (r *Repository) AddValue(ctx context.Context, v *domain.Value) error {
	const q = `
		INSERT INTO product_attribute_values (attribute_id, value, position)
		VALUES ($1, $2, $3)
		RETURNING ` + valueColumns

	created, err := scanValue(r.db.QueryRowContext(ctx, q, v.AttributeID, v.Value, v.Position))
	if err != nil {
		if platform.IsUniqueViolation(err, "product_attribute_values_unique") {
			return domain.ErrValueTaken
		}
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrAttributeNotFound
		}
		return err
	}
	*v = *created
	return nil
}

func (r *Repository) DeleteValue(ctx context.Context, attributeID, valueID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM product_attribute_values WHERE attribute_id = $1 AND id = $2`, attributeID, valueID)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrValueNotFound)
}

func (r *Repository) ValuesByAttributes(ctx context.Context, attributeIDs []uuid.UUID) (map[uuid.UUID][]domain.Value, error) {
	out := make(map[uuid.UUID][]domain.Value, len(attributeIDs))
	if len(attributeIDs) == 0 {
		return out, nil
	}
	const q = `SELECT ` + valueColumns + `
		FROM product_attribute_values
		WHERE attribute_id = ANY($1)
		ORDER BY position, value`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(uuidStrings(attributeIDs)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		v, err := scanValue(rows)
		if err != nil {
			return nil, err
		}
		out[v.AttributeID] = append(out[v.AttributeID], *v)
	}
	return out, rows.Err()
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
