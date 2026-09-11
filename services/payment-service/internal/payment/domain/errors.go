package domain

import "errors"

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrPaymentExists   = errors.New("a payment for that order already exists")
	ErrInvalidMethod   = errors.New("payment method must be CARD or COD")
	ErrInvalidAmount   = errors.New("payment amount must be positive")
	ErrNotRefundable   = errors.New("only a captured payment can be refunded")
	ErrNotSettleable   = errors.New("only a pending cash-on-delivery payment can be settled")
	ErrNotCancellable  = errors.New("only a pending payment can be cancelled")
	ErrNotOwner        = errors.New("this payment belongs to another customer")
	ErrManagerOnly     = errors.New("only a tenant ADMIN or MANAGER can view this")
)
