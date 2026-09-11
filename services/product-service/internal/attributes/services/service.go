// Package services holds the attribute-module application logic.
package services

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/attributes/domain"
	"github.com/jayedbinnazir/product-service/internal/attributes/dto"
)

var codeRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// categoryLookup is the slice of the category repo this service needs.
type categoryLookup interface {
	Exists(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
}

type Service struct {
	repo       domain.Repository
	categories categoryLookup
}

func New(repo domain.Repository, categories categoryLookup) *Service {
	return &Service{repo: repo, categories: categories}
}

// ListForCategory returns a category's attributes, each with its values.
func (s *Service) ListForCategory(ctx context.Context, tenantID, categoryID uuid.UUID) ([]domain.Detail, error) {
	attrs, err := s.repo.ListByCategory(ctx, tenantID, categoryID)
	if err != nil {
		return nil, err
	}
	return s.withValues(ctx, attrs)
}

func (s *Service) CreateAttribute(ctx context.Context, tenantID, categoryID uuid.UUID, req dto.CreateAttributeRequest) (*domain.Detail, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if !codeRE.MatchString(code) {
		return nil, domain.ErrInvalidCode
	}
	role := domain.Role(req.Role)
	if !role.Valid() {
		return nil, domain.ErrInvalidRole
	}
	ok, err := s.categories.Exists(ctx, tenantID, categoryID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrCategoryNotFound
	}

	attr := &domain.Attribute{
		TenantID:   tenantID,
		CategoryID: categoryID,
		Name:       strings.TrimSpace(req.Name),
		Code:       code,
		Role:       role,
		Position:   req.Position,
	}
	if err := s.repo.CreateAttribute(ctx, attr); err != nil {
		return nil, err
	}
	return &domain.Detail{Attribute: *attr, Values: []domain.Value{}}, nil
}

func (s *Service) UpdateAttribute(ctx context.Context, tenantID, categoryID, attributeID uuid.UUID, req dto.UpdateAttributeRequest) (*domain.Detail, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	attr, err := s.load(ctx, tenantID, categoryID, attributeID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		attr.Name = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		code := strings.ToLower(strings.TrimSpace(*req.Code))
		if !codeRE.MatchString(code) {
			return nil, domain.ErrInvalidCode
		}
		attr.Code = code
	}
	if req.Role != nil {
		role := domain.Role(*req.Role)
		if !role.Valid() {
			return nil, domain.ErrInvalidRole
		}
		attr.Role = role
	}
	if req.Position != nil {
		attr.Position = *req.Position
	}

	if err := s.repo.UpdateAttribute(ctx, attr); err != nil {
		return nil, err
	}
	details, err := s.withValues(ctx, []domain.Attribute{*attr})
	if err != nil {
		return nil, err
	}
	return &details[0], nil
}

func (s *Service) DeleteAttribute(ctx context.Context, tenantID, categoryID, attributeID uuid.UUID) error {
	if _, err := s.load(ctx, tenantID, categoryID, attributeID); err != nil {
		return err
	}
	return s.repo.DeleteAttribute(ctx, tenantID, attributeID)
}

func (s *Service) AddValue(ctx context.Context, tenantID, categoryID, attributeID uuid.UUID, req dto.AddValueRequest) (*domain.Value, error) {
	if _, err := s.load(ctx, tenantID, categoryID, attributeID); err != nil {
		return nil, err
	}
	v := &domain.Value{
		AttributeID: attributeID,
		Value:       strings.TrimSpace(req.Value),
		Position:    req.Position,
	}
	if err := s.repo.AddValue(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) DeleteValue(ctx context.Context, tenantID, categoryID, attributeID, valueID uuid.UUID) error {
	if _, err := s.load(ctx, tenantID, categoryID, attributeID); err != nil {
		return err
	}
	return s.repo.DeleteValue(ctx, attributeID, valueID)
}

// ---- helpers ----

// load fetches an attribute and confirms it belongs to the tenant and category
// named in the URL.
func (s *Service) load(ctx context.Context, tenantID, categoryID, attributeID uuid.UUID) (*domain.Attribute, error) {
	attr, err := s.repo.GetAttribute(ctx, tenantID, attributeID)
	if err != nil {
		return nil, err
	}
	if attr.CategoryID != categoryID {
		return nil, domain.ErrAttributeNotFound
	}
	return attr, nil
}

// withValues attaches each attribute's values in a single batched query.
func (s *Service) withValues(ctx context.Context, attrs []domain.Attribute) ([]domain.Detail, error) {
	ids := make([]uuid.UUID, len(attrs))
	for i := range attrs {
		ids[i] = attrs[i].ID
	}
	byAttr, err := s.repo.ValuesByAttributes(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Detail, len(attrs))
	for i := range attrs {
		values := byAttr[attrs[i].ID]
		if values == nil {
			values = []domain.Value{}
		}
		sort.SliceStable(values, func(a, b int) bool { return values[a].Position < values[b].Position })
		out[i] = domain.Detail{Attribute: attrs[i], Values: values}
	}
	return out, nil
}
