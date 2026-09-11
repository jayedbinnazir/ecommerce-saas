package domain

import "errors"

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrSlugTaken        = errors.New("a category with that slug already exists")
	ErrInvalidSlug      = errors.New("slug must be lowercase alphanumeric words separated by hyphens")
	ErrNoUpdateFields   = errors.New("no fields to update")
	ErrParentCycle      = errors.New("a category cannot be its own ancestor")
)
