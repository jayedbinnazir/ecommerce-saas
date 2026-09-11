package domain

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidType          = errors.New("notification type is required")
)
