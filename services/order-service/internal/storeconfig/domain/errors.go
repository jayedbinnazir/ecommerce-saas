package domain

import "errors"

var (
	ErrShippingRateNotFound = errors.New("shipping rate not found")
	ErrShippingRateInactive = errors.New("that shipping rate is not available")

	ErrCouponNotFound    = errors.New("coupon not found")
	ErrCouponCodeTaken   = errors.New("a coupon with that code already exists")
	ErrInvalidCouponKind = errors.New("coupon kind must be PERCENT or FIXED")
	ErrCouponInactive    = errors.New("that coupon is no longer active")
	ErrCouponNotStarted  = errors.New("that coupon is not valid yet")
	ErrCouponExpired     = errors.New("that coupon has expired")
	ErrCouponExhausted   = errors.New("that coupon has reached its redemption limit")
	ErrCouponMinOrder    = errors.New("your order does not meet this coupon's minimum")

	ErrNoUpdateFields = errors.New("no fields to update")
	ErrInvalidTaxRate = errors.New("tax rate must be between 0 and 10000 basis points")
)
