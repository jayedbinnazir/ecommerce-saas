package domain

import "errors"

var (
	ErrStockItemNotFound    = errors.New("no stock item for that SKU")
	ErrStockItemExists      = errors.New("a stock item for that SKU already exists")
	ErrInvalidSKU           = errors.New("sku must be 1-100 chars of letters, digits, dot, underscore or hyphen")
	ErrInvalidQuantity      = errors.New("quantity must be a positive whole number")
	ErrNoUpdateFields       = errors.New("no fields to update")
	ErrInsufficientStock    = errors.New("not enough available stock")
	ErrInsufficientReserved = errors.New("not enough reserved stock")
	ErrWouldGoNegative      = errors.New("that change would take on-hand below zero or below what is reserved")
)
