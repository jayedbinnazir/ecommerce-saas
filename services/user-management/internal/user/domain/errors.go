package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrEmptyUserName      = errors.New("user name must not be empty")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoUpdateFields     = errors.New("no fields to update")

	ErrAddressNotFound = errors.New("address not found")
	ErrAddressNotOwned = errors.New("address does not belong to user")
)
