package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/repository"
)

// AddressService owns the "at most one default shipping/billing address per
// user" invariant, so mutations that touch a default flag run in a transaction.
type AddressService struct {
	db    *sql.DB
	users domain.UserRepository
}

func NewAddressService(db *sql.DB, users domain.UserRepository) *AddressService {
	return &AddressService{db: db, users: users}
}

func (s *AddressService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateAddressRequest) (*domain.Address, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return nil, err
	}

	addr := &domain.Address{
		UserID:            userID,
		Label:             req.Label,
		RecipientName:     req.RecipientName,
		Phone:             req.Phone,
		AddressLine1:      req.AddressLine1,
		AddressLine2:      req.AddressLine2,
		City:              req.City,
		State:             req.State,
		PostalCode:        req.PostalCode,
		Country:           req.Country,
		IsDefaultShipping: req.IsDefaultShipping,
		IsDefaultBilling:  req.IsDefaultBilling,
	}

	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		repo := repository.NewAddressRepository(tx)
		if err := clearDefaults(ctx, repo, userID, addr.IsDefaultShipping, addr.IsDefaultBilling); err != nil {
			return err
		}
		return repo.Create(ctx, addr)
	})
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *AddressService) Get(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error) {
	addr, err := repository.NewAddressRepository(s.db).GetByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr.UserID != userID {
		return nil, domain.ErrAddressNotOwned
	}
	return addr, nil
}

func (s *AddressService) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return nil, err
	}
	return repository.NewAddressRepository(s.db).ListByUser(ctx, userID)
}

func (s *AddressService) Update(ctx context.Context, userID, addressID uuid.UUID, req dto.UpdateAddressRequest) (*domain.Address, error) {
	var updated *domain.Address

	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		repo := repository.NewAddressRepository(tx)

		addr, err := repo.GetByID(ctx, addressID)
		if err != nil {
			return err
		}
		if addr.UserID != userID {
			return domain.ErrAddressNotOwned
		}

		applyAddressPatch(addr, req)

		if err := clearDefaults(ctx, repo, userID, addr.IsDefaultShipping, addr.IsDefaultBilling); err != nil {
			return err
		}
		if err := repo.Update(ctx, addr); err != nil {
			return err
		}
		updated = addr
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *AddressService) Delete(ctx context.Context, userID, addressID uuid.UUID) error {
	repo := repository.NewAddressRepository(s.db)
	addr, err := repo.GetByID(ctx, addressID)
	if err != nil {
		return err
	}
	if addr.UserID != userID {
		return domain.ErrAddressNotOwned
	}
	return repo.Delete(ctx, addressID)
}

func clearDefaults(ctx context.Context, repo *repository.AddressRepository, userID uuid.UUID, shipping, billing bool) error {
	if shipping {
		if err := repo.ClearDefaultShipping(ctx, userID); err != nil {
			return err
		}
	}
	if billing {
		if err := repo.ClearDefaultBilling(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

func applyAddressPatch(a *domain.Address, req dto.UpdateAddressRequest) {
	if req.Label != nil {
		a.Label = req.Label
	}
	if req.RecipientName != nil {
		a.RecipientName = *req.RecipientName
	}
	if req.Phone != nil {
		a.Phone = *req.Phone
	}
	if req.AddressLine1 != nil {
		a.AddressLine1 = *req.AddressLine1
	}
	if req.AddressLine2 != nil {
		a.AddressLine2 = req.AddressLine2
	}
	if req.City != nil {
		a.City = *req.City
	}
	if req.State != nil {
		a.State = req.State
	}
	if req.PostalCode != nil {
		a.PostalCode = req.PostalCode
	}
	if req.Country != nil {
		a.Country = *req.Country
	}
	if req.IsDefaultShipping != nil {
		a.IsDefaultShipping = *req.IsDefaultShipping
	}
	if req.IsDefaultBilling != nil {
		a.IsDefaultBilling = *req.IsDefaultBilling
	}
}
