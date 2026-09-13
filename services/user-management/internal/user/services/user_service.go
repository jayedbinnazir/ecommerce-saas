// Package services holds the user-module application logic. Credential handling
// and account provisioning live in the auth module; this service covers profile
// reads/writes and platform-admin user management.
package services

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/dto"
)

type UserService struct {
	repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	return s.repo.List(ctx, platform.Page{Limit: limit, Offset: offset})
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (*domain.User, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}

	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domain.ErrEmptyUserName
		}
		user.Name = name
	}
	if req.Email != nil {
		email := NormalizeEmail(*req.Email)
		if email == "" {
			return nil, domain.ErrInvalidEmail
		}
		user.Email = email
	}
	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		if phone == "" {
			user.Phone = nil
		} else {
			user.Phone = &phone
		}
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// NormalizeEmail lower-cases and trims an email; shared with the auth module.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
