package domain

import (
	"context"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context, page platform.Page) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AddressRepository interface {
	Create(ctx context.Context, address *Address) error
	GetByID(ctx context.Context, id uuid.UUID) (*Address, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Address, error)
	Update(ctx context.Context, address *Address) error
	Delete(ctx context.Context, id uuid.UUID) error
	ClearDefaultShipping(ctx context.Context, userID uuid.UUID) error
	ClearDefaultBilling(ctx context.Context, userID uuid.UUID) error
}
