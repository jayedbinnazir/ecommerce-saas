package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

type ImageRepository struct {
	db platform.DBTX
}

func NewImageRepository(db platform.DBTX) *ImageRepository {
	return &ImageRepository{db: db}
}

var _ domain.ImageRepository = (*ImageRepository)(nil)

const imageColumns = `id, product_id, url, storage_key, alt, position, created_at`

func scanImage(row platform.Scanner) (*domain.Image, error) {
	var img domain.Image
	if err := row.Scan(&img.ID, &img.ProductID, &img.URL, &img.StorageKey, &img.Alt, &img.Position, &img.CreatedAt); err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *ImageRepository) Create(ctx context.Context, img *domain.Image) error {
	const q = `
		INSERT INTO product_images (id, product_id, url, storage_key, alt, position)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + imageColumns

	created, err := scanImage(r.db.QueryRowContext(ctx, q, img.ID, img.ProductID, img.URL, img.StorageKey, img.Alt, img.Position))
	if err != nil {
		if platform.IsUniqueViolation(err, "product_images_storage_key_key") {
			return domain.ErrImageAlreadyExists
		}
		return err
	}
	*img = *created
	return nil
}

func (r *ImageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	img, err := scanImage(r.db.QueryRowContext(ctx, `SELECT `+imageColumns+` FROM product_images WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrImageNotFound
	}
	return img, err
}

func (r *ImageRepository) ListByProduct(ctx context.Context, productID uuid.UUID) ([]domain.Image, error) {
	const q = `SELECT ` + imageColumns + ` FROM product_images WHERE product_id = $1 ORDER BY position, created_at`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Image, 0)
	for rows.Next() {
		img, err := scanImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *img)
	}
	return out, rows.Err()
}

func (r *ImageRepository) Update(ctx context.Context, img *domain.Image) error {
	const q = `UPDATE product_images SET url = $2, alt = $3, position = $4 WHERE id = $1 RETURNING ` + imageColumns
	updated, err := scanImage(r.db.QueryRowContext(ctx, q, img.ID, img.URL, img.Alt, img.Position))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrImageNotFound
	}
	if err != nil {
		return err
	}
	*img = *updated
	return nil
}

func (r *ImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM product_images WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrImageNotFound)
}
