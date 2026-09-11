package domain

import "errors"

var (
	ErrCartNotFound        = errors.New("no active cart")
	ErrItemNotFound        = errors.New("that item is not in the cart")
	ErrInvalidSKU          = errors.New("sku must be 1-100 chars of letters, digits, dot, underscore or hyphen")
	ErrInvalidQuantity     = errors.New("quantity must be a positive whole number")
	ErrNoUpdateFields      = errors.New("no fields to update")
	ErrVariantNotAvailable = errors.New("that product is not available for purchase")
	ErrInsufficientStock   = errors.New("not enough stock for the requested quantity")
	ErrCurrencyMismatch    = errors.New("every item in a cart must use the same currency")
)
