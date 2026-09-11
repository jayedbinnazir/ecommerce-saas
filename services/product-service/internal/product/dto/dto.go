// Package dto holds the request/response payloads for the product module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/product/domain"
)

// ---- requests: product ----

type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required,max=255"`
	Slug        string  `json:"slug" binding:"required,min=2,max=200"`
	Description *string `json:"description" binding:"omitempty,max=20000"`
}

type UpdateProductRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Slug        *string `json:"slug" binding:"omitempty,min=2,max=200"`
	Description *string `json:"description" binding:"omitempty,max=20000"`
	Status      *string `json:"status" binding:"omitempty,oneof=DRAFT ACTIVE ARCHIVED"`
}

func (r UpdateProductRequest) IsEmpty() bool {
	return r.Name == nil && r.Slug == nil && r.Description == nil && r.Status == nil
}

type SetCategoriesRequest struct {
	CategoryIDs []string `json:"category_ids" binding:"dive,uuid"`
}

// ---- requests: variant ----

// VariantOptionInput picks one value for one variant axis (attribute of role
// VARIANT). A full set of these identifies the combination this variant is.
type VariantOptionInput struct {
	AttributeID string `json:"attribute_id" binding:"required,uuid"`
	ValueID     string `json:"value_id" binding:"required,uuid"`
}

type CreateVariantRequest struct {
	SKU                 string               `json:"sku" binding:"required,max=100"`
	Title               *string              `json:"title" binding:"omitempty,max=255"`
	PriceCents          int64                `json:"price_cents" binding:"gte=0"`
	Currency            *string              `json:"currency" binding:"omitempty,len=3,uppercase"`
	CompareAtPriceCents *int64               `json:"compare_at_price_cents" binding:"omitempty,gte=0"`
	WeightGrams         *int                 `json:"weight_grams" binding:"omitempty,gte=0"`
	Barcode             *string              `json:"barcode" binding:"omitempty,max=100"`
	Position            int                  `json:"position"`
	IsDefault           bool                 `json:"is_default"`
	Options             []VariantOptionInput `json:"options" binding:"omitempty,dive"`
}

type UpdateVariantRequest struct {
	SKU                 *string `json:"sku" binding:"omitempty,max=100"`
	Title               *string `json:"title" binding:"omitempty,max=255"`
	PriceCents          *int64  `json:"price_cents" binding:"omitempty,gte=0"`
	Currency            *string `json:"currency" binding:"omitempty,len=3,uppercase"`
	CompareAtPriceCents *int64  `json:"compare_at_price_cents" binding:"omitempty,gte=0"`
	WeightGrams         *int    `json:"weight_grams" binding:"omitempty,gte=0"`
	Barcode             *string `json:"barcode" binding:"omitempty,max=100"`
	Position            *int    `json:"position"`
	IsDefault           *bool   `json:"is_default"`
	// Options nil = leave as-is; [] = clear; non-empty = replace.
	Options []VariantOptionInput `json:"options" binding:"omitempty,dive"`
}

func (r UpdateVariantRequest) IsEmpty() bool {
	return r.SKU == nil && r.Title == nil && r.PriceCents == nil && r.Currency == nil &&
		r.CompareAtPriceCents == nil && r.WeightGrams == nil && r.Barcode == nil &&
		r.Position == nil && r.IsDefault == nil && r.Options == nil
}

// SetSpecsRequest replaces a product's whole set of display-spec values.
type SetSpecsRequest struct {
	Specs []SpecInput `json:"specs" binding:"dive"`
}

type SpecInput struct {
	AttributeID string `json:"attribute_id" binding:"required,uuid"`
	Value       string `json:"value" binding:"required,max=500"`
}

// ---- requests: image (S3 direct upload) ----

// ImageUploadURLRequest asks for a presigned URL to PUT the raw file to.
type ImageUploadURLRequest struct {
	ContentType string `json:"content_type" binding:"required,oneof=image/jpeg image/png image/webp image/avif"`
}

type ImageUploadURLResponse struct {
	ImageID    uuid.UUID `json:"image_id"`
	StorageKey string    `json:"storage_key"`
	UploadURL  string    `json:"upload_url"` // PUT the file here with the same Content-Type
	PublicURL  string    `json:"public_url"` // where the image will be served once confirmed
	ExpiresIn  int       `json:"expires_in"` // seconds the upload URL is valid
}

// ConfirmImageRequest registers a successfully uploaded object as a product image.
type ConfirmImageRequest struct {
	StorageKey string  `json:"storage_key" binding:"required"`
	Alt        *string `json:"alt" binding:"omitempty,max=255"`
	Position   int     `json:"position"`
}

type UpdateImageRequest struct {
	Alt      *string `json:"alt" binding:"omitempty,max=255"`
	Position *int    `json:"position"`
}

func (r UpdateImageRequest) IsEmpty() bool {
	return r.Alt == nil && r.Position == nil
}

// ---- responses ----

type ProductResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func FromProduct(p *domain.Product) ProductResponse {
	return ProductResponse{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func FromProducts(ps []domain.Product) []ProductResponse {
	out := make([]ProductResponse, len(ps))
	for i := range ps {
		out[i] = FromProduct(&ps[i])
	}
	return out
}

type VariantResponse struct {
	ID                  uuid.UUID         `json:"id"`
	ProductID           uuid.UUID         `json:"product_id"`
	SKU                 string            `json:"sku"`
	Title               *string           `json:"title,omitempty"`
	PriceCents          int64             `json:"price_cents"`
	Currency            string            `json:"currency"`
	CompareAtPriceCents *int64            `json:"compare_at_price_cents,omitempty"`
	WeightGrams         *int              `json:"weight_grams,omitempty"`
	Barcode             *string           `json:"barcode,omitempty"`
	Position            int               `json:"position"`
	IsDefault           bool              `json:"is_default"`
	Options             map[string]string `json:"options,omitempty"` // {attribute code: chosen value}
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

func FromVariant(v *domain.Variant) VariantResponse {
	return VariantResponse{
		ID:                  v.ID,
		ProductID:           v.ProductID,
		SKU:                 v.SKU,
		Title:               v.Title,
		PriceCents:          v.PriceCents,
		Currency:            v.Currency,
		CompareAtPriceCents: v.CompareAtPriceCents,
		WeightGrams:         v.WeightGrams,
		Barcode:             v.Barcode,
		Position:            v.Position,
		IsDefault:           v.IsDefault,
		CreatedAt:           v.CreatedAt,
		UpdatedAt:           v.UpdatedAt,
	}
}

func FromVariants(vs []domain.Variant) []VariantResponse {
	out := make([]VariantResponse, len(vs))
	for i := range vs {
		out[i] = FromVariant(&vs[i])
	}
	return out
}

type ImageResponse struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	URL       string    `json:"url"`
	Alt       *string   `json:"alt,omitempty"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

func FromImage(img *domain.Image) ImageResponse {
	return ImageResponse{
		ID:        img.ID,
		ProductID: img.ProductID,
		URL:       img.URL,
		Alt:       img.Alt,
		Position:  img.Position,
		CreatedAt: img.CreatedAt,
	}
}

func FromImages(imgs []domain.Image) []ImageResponse {
	out := make([]ImageResponse, len(imgs))
	for i := range imgs {
		out[i] = FromImage(&imgs[i])
	}
	return out
}

// OptionValueResponse / OptionAxisResponse describe the customer-selectable axes.
type OptionValueResponse struct {
	ID    uuid.UUID `json:"id"`
	Value string    `json:"value"`
}

type OptionAxisResponse struct {
	Code   string                `json:"code"`
	Name   string                `json:"name"`
	Values []OptionValueResponse `json:"values"`
}

// SpecResponse is one display-only specification line.
type SpecResponse struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProductDetailResponse struct {
	ProductResponse
	Variants    []VariantResponse    `json:"variants"`
	Images      []ImageResponse      `json:"images"`
	CategoryIDs []uuid.UUID          `json:"category_ids"`
	Options     []OptionAxisResponse `json:"options"` // pick one value per axis -> a variant
	Specs       []SpecResponse       `json:"specs"`   // display only
}

func FromDetail(d *domain.Detail) ProductDetailResponse {
	ids := d.CategoryID
	if ids == nil {
		ids = []uuid.UUID{}
	}

	variants := make([]VariantResponse, len(d.Variants))
	for i := range d.Variants {
		vr := FromVariant(&d.Variants[i])
		if opts := d.VariantOptions[d.Variants[i].ID]; len(opts) > 0 {
			vr.Options = opts
		}
		variants[i] = vr
	}

	options := make([]OptionAxisResponse, 0, len(d.Options))
	for _, axis := range d.Options {
		values := make([]OptionValueResponse, len(axis.Values))
		for j, v := range axis.Values {
			values[j] = OptionValueResponse{ID: v.ID, Value: v.Value}
		}
		options = append(options, OptionAxisResponse{Code: axis.Code, Name: axis.Name, Values: values})
	}

	specs := make([]SpecResponse, 0, len(d.Specs))
	for _, s := range d.Specs {
		specs = append(specs, SpecResponse{Code: s.Code, Name: s.Name, Value: s.Value})
	}

	return ProductDetailResponse{
		ProductResponse: FromProduct(&d.Product),
		Variants:        variants,
		Images:          FromImages(d.Images),
		CategoryIDs:     ids,
		Options:         options,
		Specs:           specs,
	}
}
