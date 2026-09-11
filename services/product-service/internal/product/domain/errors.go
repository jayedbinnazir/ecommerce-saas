package domain

import "errors"

var (
	ErrProductNotFound        = errors.New("product not found")
	ErrVariantNotFound        = errors.New("variant not found")
	ErrImageNotFound          = errors.New("image not found")
	ErrImageNotUploaded       = errors.New("no uploaded file found at that storage key")
	ErrImageKeyMismatch       = errors.New("storage key does not belong to this product")
	ErrImageAlreadyExists     = errors.New("that image has already been added")
	ErrUnsupportedImageType   = errors.New("unsupported image type (use jpeg, png, webp or avif)")
	ErrSlugTaken              = errors.New("a product with that slug already exists")
	ErrSkuTaken               = errors.New("a variant with that SKU already exists")
	ErrInvalidSlug            = errors.New("slug must be lowercase alphanumeric words separated by hyphens")
	ErrInvalidStatus          = errors.New("invalid product status")
	ErrNoUpdateFields         = errors.New("no fields to update")
	ErrActivateNeedsVariant   = errors.New("a product needs at least one variant before it can be activated")
	ErrVariantNotInProduct    = errors.New("that variant does not belong to this product")
	ErrCategoryNotInTenant    = errors.New("one or more categories do not belong to this tenant")
	ErrAttributeNotForProduct = errors.New("that attribute is not defined for this product's categories, or has the wrong role")
	ErrOptionValueInvalid     = errors.New("an option value does not belong to its attribute")
)
