// Package services holds the role-module application logic.
package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/dto"
)

type RoleService struct {
	repo domain.RoleRepository
}

func NewRoleService(repo domain.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Create(ctx context.Context, req dto.CreateRoleRequest) (*domain.Role, error) {
	name := domain.RoleName(req.Name)
	if !name.Valid() {
		return nil, domain.ErrInvalidRoleName
	}
	role := &domain.Role{Name: name, Description: trimPtr(req.Description)}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) Get(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RoleService) List(ctx context.Context, limit, offset int) ([]domain.Role, error) {
	return s.repo.List(ctx, platform.Page{Limit: limit, Offset: offset})
}

func (s *RoleService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateRoleRequest) (*domain.Role, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name := domain.RoleName(*req.Name)
		if !name.Valid() {
			return nil, domain.ErrInvalidRoleName
		}
		role.Name = name
	}
	if req.Description != nil {
		role.Description = trimPtr(req.Description)
	}
	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
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
