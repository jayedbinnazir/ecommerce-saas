package domain

import "errors"

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrNotOwner             = errors.New("this order belongs to another customer")
	ErrManagerOnly          = errors.New("only a tenant ADMIN or MANAGER can do that")
	ErrInvalidTransition    = errors.New("the order cannot move to that state from its current one")
	ErrInvalidPaymentMethod = errors.New("payment method must be CARD or COD")
)
