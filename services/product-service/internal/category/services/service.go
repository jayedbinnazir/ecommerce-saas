// Package services holds the category-module application logic.
package services

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/category/domain"
	"github.com/jayedbinnazir/product-service/internal/category/dto"
)

var slugRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, req dto.CreateCategoryRequest) (*domain.Category, error) {
	slug := normalizeSlug(req.Slug)
	if !slugRE.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	category := &domain.Category{
		TenantID: tenantID,
		Name:     strings.TrimSpace(req.Name),
		Slug:     slug,
		Position: req.Position,
	}
	if req.ParentID != nil && *req.ParentID != "" {
		parentID := uuid.MustParse(*req.ParentID)
		category.ParentID = &parentID
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateCategoryRequest) (*domain.Category, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}

	category, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		category.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		slug := normalizeSlug(*req.Slug)
		if !slugRE.MatchString(slug) {
			return nil, domain.ErrInvalidSlug
		}
		category.Slug = slug
	}
	if req.Position != nil {
		category.Position = *req.Position
	}
	if req.ParentID != nil {
		if *req.ParentID == "" {
			category.ParentID = nil
		} else {
			parentID := uuid.MustParse(*req.ParentID)
			if parentID == id {
				return nil, domain.ErrParentCycle
			}
			category.ParentID = &parentID
		}
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeSlug(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
