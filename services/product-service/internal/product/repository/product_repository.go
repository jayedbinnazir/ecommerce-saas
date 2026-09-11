// Package repository holds the database/sql implementations of the product-module
// repositories.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

type ProductRepository struct {
	db platform.DBTX
}

func NewProductRepository(db platform.DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

var _ domain.ProductRepository = (*ProductRepository)(nil)

const productColumns = `id, tenant_id, name, slug, description, status, created_at, updated_at`

func scanProduct(row platform.Scanner) (*domain.Product, error) {
	var p domain.Product
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	const q = `
		INSERT INTO products (tenant_id, name, slug, description, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + productColumns

	created, err := scanProduct(r.db.QueryRowContext(ctx, q, p.TenantID, p.Name, p.Slug, p.Description, p.Status))
	if err != nil {
		if platform.IsUniqueViolation(err, "products_tenant_slug_key") {
			return domain.ErrSlugTaken
		}
		return err
	}
	*p = *created
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	const q = `SELECT ` + productColumns + ` FROM products WHERE tenant_id = $1 AND id = $2`
	p, err := scanProduct(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	return p, err
}

func (r *ProductRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Product, error) {
	const q = `SELECT ` + productColumns + ` FROM products WHERE tenant_id = $1 AND slug = $2`
	p, err := scanProduct(r.db.QueryRowContext(ctx, q, tenantID, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	return p, err
}

func (r *ProductRepository) List(ctx context.Context, f domain.ListFilter) ([]domain.Product, error) {
	var (
		where = []string{"p.tenant_id = $1"}
		args  = []any{f.TenantID}
	)

	if f.Status != nil {
		args = append(args, *f.Status)
		where = append(where, "p.status = $"+itoa(len(args)))
	}
	if f.Search != "" {
		// full-text over name + description (products.search is a generated tsvector)
		args = append(args, f.Search)
		where = append(where, "p.search @@ websearch_to_tsquery('simple', $"+itoa(len(args))+")")
	}

	join := ""
	if f.CategoryID != nil {
		join = " JOIN product_categories pc ON pc.product_id = p.id"
		args = append(args, *f.CategoryID)
		where = append(where, "pc.category_id = $"+itoa(len(args)))
	}

	args = append(args, f.Page.Limit, f.Page.Offset)
	q := `SELECT ` + prefixed(productColumns, "p") + ` FROM products p` + join +
		` WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY p.created_at DESC LIMIT $` + itoa(len(args)-1) + ` OFFSET $` + itoa(len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	const q = `
		UPDATE products SET name = $3, slug = $4, description = $5, status = $6
		WHERE tenant_id = $1 AND id = $2
		RETURNING ` + productColumns

	updated, err := scanProduct(r.db.QueryRowContext(ctx, q, p.TenantID, p.ID, p.Name, p.Slug, p.Description, p.Status))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrProductNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "products_tenant_slug_key") {
			return domain.ErrSlugTaken
		}
		return err
	}
	*p = *updated
	return nil
}

// Exists reports whether product id belongs to tenantID.
func (r *ProductRepository) Exists(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM products WHERE tenant_id = $1 AND id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, tenantID, id).Scan(&exists)
	return exists, err
}

func (r *ProductRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrProductNotFound)
}

// small helpers so the dynamic query above stays readable

func itoa(n int) string { return strconv.Itoa(n) }

func prefixed(columns, alias string) string {
	parts := strings.Split(columns, ", ")
	for i, p := range parts {
		parts[i] = alias + "." + p
	}
	return strings.Join(parts, ", ")
}
