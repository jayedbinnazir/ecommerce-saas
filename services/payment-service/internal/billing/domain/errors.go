package domain

import "errors"

var (
	ErrPlanNotFound         = errors.New("plan not found")
	ErrPlanInactive         = errors.New("plan is not available")
	ErrSubscriptionNotFound = errors.New("no active subscription")
	ErrAlreadySubscribed    = errors.New("user already has an active subscription")
)
