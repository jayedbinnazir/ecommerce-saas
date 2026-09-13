// Package services holds the permission-module application logic.
package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/permission/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/permission/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

// =====================================================================
// Permission catalog
// =====================================================================

type PermissionService struct {
	repo domain.PermissionRepository
}

func NewPermissionService(repo domain.PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) Create(ctx context.Context, req dto.CreatePermissionRequest) (*domain.Permission, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidPermissionName
	}
	p := &domain.Permission{Name: name, Description: trimPtr(req.Description)}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PermissionService) Get(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PermissionService) List(ctx context.Context, limit, offset int) ([]domain.Permission, error) {
	return s.repo.List(ctx, platform.Page{Limit: limit, Offset: offset})
}

func (s *PermissionService) Update(ctx context.Context, id uuid.UUID, req dto.UpdatePermissionRequest) (*domain.Permission, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrInvalidPermissionName
		}
		p.Name = name
	}
	if req.Description != nil {
		p.Description = trimPtr(req.Description)
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *PermissionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// =====================================================================
// Grants
// =====================================================================

type GrantService struct {
	grants domain.GrantRepository
}

func NewGrantService(grants domain.GrantRepository) *GrantService {
	return &GrantService{grants: grants}
}

func (s *GrantService) AssignToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.grants.AssignToRole(ctx, roleID, permissionID)
}

func (s *GrantService) RevokeFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return s.grants.RevokeFromRole(ctx, roleID, permissionID)
}

func (s *GrantService) ListByRole(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	return s.grants.ListByRole(ctx, roleID)
}

func (s *GrantService) AssignToMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error {
	return s.grants.AssignToMembership(ctx, membershipID, permissionID)
}

func (s *GrantService) RevokeFromMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error {
	return s.grants.RevokeFromMembership(ctx, membershipID, permissionID)
}

func (s *GrantService) ListByMembership(ctx context.Context, membershipID uuid.UUID) ([]domain.Permission, error) {
	return s.grants.ListByMembership(ctx, membershipID)
}

// EffectiveForMembership returns the member's full permission-name set.
func (s *GrantService) EffectiveForMembership(ctx context.Context, membershipID uuid.UUID) ([]string, error) {
	return s.grants.EffectiveForMembership(ctx, membershipID)
}

// HasPermission reports whether the member holds perm (role or individual grant).
func (s *GrantService) HasPermission(ctx context.Context, membershipID uuid.UUID, perm string) (bool, error) {
	names, err := s.grants.EffectiveForMembership(ctx, membershipID)
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == perm {
			return true, nil
		}
	}
	return false, nil
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
