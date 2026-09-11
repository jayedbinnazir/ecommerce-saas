package domain

import "errors"

var (
	ErrAttributeNotFound = errors.New("attribute not found")
	ErrValueNotFound     = errors.New("attribute value not found")
	ErrInvalidRole       = errors.New("role must be VARIANT or SPEC")
	ErrInvalidCode       = errors.New("code must be lowercase alphanumeric words separated by hyphens")
	ErrCodeTaken         = errors.New("an attribute with that code already exists in this category")
	ErrValueTaken        = errors.New("that value already exists for this attribute")
	ErrNoUpdateFields    = errors.New("no fields to update")
	ErrCategoryNotFound  = errors.New("category not found")
)
