package domain

import "errors"

var (
	ErrReturnNotFound         = errors.New("return not found")
	ErrNotOwner               = errors.New("this order belongs to another customer")
	ErrManagerOnly            = errors.New("only a tenant ADMIN or MANAGER can do that")
	ErrOrderNotReturnable     = errors.New("only a fulfilled order can be returned")
	ErrAlreadyResolved        = errors.New("this return has already been resolved")
	ErrInvalidReturnItems     = errors.New("return items must match the order's lines and quantities")
	ErrNoReturnItems          = errors.New("a return needs at least one item")
	ErrReturnQuantityExceeded = errors.New("that quantity has already been returned or is pending return")
)
