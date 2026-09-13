// Package services holds the tenant-module application logic, including the
// "register your store" onboarding flow.
package services

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/repository"
)

var slugRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Service struct {
	db           *sql.DB
	repo         domain.Repository
	subscription domain.SubscriptionGuard
	enroller     domain.OwnerEnroller
}

func New(
	db *sql.DB,
	repo domain.Repository,
	subscription domain.SubscriptionGuard,
	enroller domain.OwnerEnroller,
) *Service {
	return &Service{db: db, repo: repo, subscription: subscription, enroller: enroller}
}

// Create runs the onboarding flow: verify the caller has an active subscription,
// then create the store and enroll them as its ADMIN in one transaction.
func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, req dto.CreateTenantRequest) (*domain.Tenant, error) {
	if ownerID == uuid.Nil {
		return nil, domain.ErrOwnerRequired
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRE.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	if err := s.subscription.RequireActiveSubscription(ctx, ownerID); err != nil {
		return nil, err
	}

	tenant := &domain.Tenant{
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Status:      domain.StatusActive,
		OwnerUserID: ownerID,
	}

	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := repository.New(tx).Create(ctx, tenant); err != nil {
			return err
		}
		return s.enroller.EnrollOwnerAsAdmin(ctx, tx, tenant.ID, ownerID)
	})
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListMine(ctx context.Context, ownerID uuid.UUID) ([]domain.Tenant, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]domain.Tenant, error) {
	return s.repo.List(ctx, platform.Page{Limit: limit, Offset: offset})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req dto.UpdateTenantRequest) (*domain.Tenant, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		tenant.Name = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		status := domain.Status(*req.Status)
		if !status.Valid() {
			return nil, domain.ErrInvalidStatus
		}
		tenant.Status = status
	}
	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
