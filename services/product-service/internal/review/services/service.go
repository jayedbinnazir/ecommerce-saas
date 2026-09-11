// Package services holds the review-module logic.
package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/review/domain"
	"github.com/jayedbinnazir/product-service/internal/review/dto"
)

type Service struct {
	repo domain.Repository
	db   productLookup
}

// productLookup is the slice of the product repo the review service needs.
type productLookup interface {
	Exists(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}

// New wires the review service. products is used to confirm the product belongs
// to the tenant before a review is accepted.
func New(repo domain.Repository, products productLookup) *Service {
	return &Service{repo: repo, db: products}
}

func (s *Service) Create(ctx context.Context, tenantID, productID, customerID uuid.UUID, req dto.CreateReviewRequest) (*domain.Review, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, domain.ErrInvalidRating
	}
	ok, err := s.db.Exists(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrReviewNotFound
	}

	rv := &domain.Review{
		TenantID:   tenantID,
		ProductID:  productID,
		CustomerID: customerID,
		Rating:     req.Rating,
		Title:      trim(req.Title),
		Body:       trim(req.Body),
	}
	if err := s.repo.Upsert(ctx, rv); err != nil {
		return nil, err
	}
	return rv, nil
}

func (s *Service) List(ctx context.Context, productID uuid.UUID, page platform.Page) ([]domain.Review, error) {
	return s.repo.ListByProduct(ctx, productID, page)
}

func (s *Service) Summary(ctx context.Context, productID uuid.UUID) (domain.Summary, error) {
	return s.repo.Summary(ctx, productID)
}

// Delete removes a review — only the author may.
func (s *Service) Delete(ctx context.Context, tenantID, reviewID, requesterID uuid.UUID) error {
	rv, err := s.repo.GetByID(ctx, tenantID, reviewID)
	if err != nil {
		return err
	}
	if rv.CustomerID != requesterID {
		return domain.ErrNotAuthor
	}
	return s.repo.Delete(ctx, tenantID, reviewID)
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
