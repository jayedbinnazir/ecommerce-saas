package domain

import "errors"

var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrSlugTaken      = errors.New("tenant slug already taken")
	ErrInvalidSlug    = errors.New("slug must be lowercase alphanumeric words separated by hyphens")
	ErrInvalidStatus  = errors.New("invalid tenant status")
	ErrNoUpdateFields = errors.New("no fields to update")
	ErrOwnerRequired  = errors.New("tenant owner is required")
	ErrForbidden      = errors.New("not allowed to manage this tenant")
)
