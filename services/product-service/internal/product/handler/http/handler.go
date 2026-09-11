// Package httphandler contains the Gin HTTP handlers for the product module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/httpx"
	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
	"github.com/jayedbinnazir/product-service/internal/product/dto"
	"github.com/jayedbinnazir/product-service/internal/product/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// ---------------------------------------------------------------------
// Storefront (public) — ACTIVE products only
// ---------------------------------------------------------------------

func (h *Handler) ListPublished(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)

	active := domain.StatusActive
	filter := domain.ListFilter{
		TenantID: tenantID,
		Status:   &active,
		Search:   c.Query("q"),
		Page:     platform.Page{Limit: limit, Offset: offset},
	}
	if cat := c.Query("category"); cat != "" {
		id, err := uuid.Parse(cat)
		if err != nil {
			_ = c.Error(httpx.BadRequest("query parameter category must be a UUID"))
			return
		}
		filter.CategoryID = &id
	}

	products, err := h.svc.ListProducts(c.Request.Context(), filter)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromProducts(products), limit, offset)
}

func (h *Handler) GetPublished(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	productID, ok := httpx.UUIDParam(c, "productId")
	if !ok {
		return
	}
	detail, err := h.svc.GetPublishedProduct(c.Request.Context(), tenantID, productID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// ---------------------------------------------------------------------
// Management (tenant ADMIN / MANAGER) — any status
// ---------------------------------------------------------------------

func (h *Handler) List(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)

	filter := domain.ListFilter{
		TenantID: tenantID,
		Search:   c.Query("q"),
		Page:     platform.Page{Limit: limit, Offset: offset},
	}
	if st := c.Query("status"); st != "" {
		status := domain.Status(st)
		if !status.Valid() {
			_ = c.Error(httpx.BadRequest("query parameter status must be DRAFT, ACTIVE or ARCHIVED"))
			return
		}
		filter.Status = &status
	}
	if cat := c.Query("category"); cat != "" {
		id, err := uuid.Parse(cat)
		if err != nil {
			_ = c.Error(httpx.BadRequest("query parameter category must be a UUID"))
			return
		}
		filter.CategoryID = &id
	}

	products, err := h.svc.ListProducts(c.Request.Context(), filter)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromProducts(products), limit, offset)
}

func (h *Handler) Get(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetProduct(c.Request.Context(), tenantID, productID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) Create(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreateProductRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.CreateProduct(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromDetail(detail))
}

func (h *Handler) Update(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.UpdateProductRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.UpdateProduct(c.Request.Context(), tenantID, productID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) Delete(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProduct(c.Request.Context(), tenantID, productID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) SetCategories(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.SetCategoriesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	ids, err := h.svc.SetCategories(c.Request.Context(), tenantID, productID, req.CategoryIDs)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, gin.H{"category_ids": ids})
}

// SetSpecs — PUT .../products/:productId/specs  → replace the display-spec values.
func (h *Handler) SetSpecs(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.SetSpecsRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.SetSpecs(c.Request.Context(), tenantID, productID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// ---- variants ----

func (h *Handler) AddVariant(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.CreateVariantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	variant, err := h.svc.AddVariant(c.Request.Context(), tenantID, productID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromVariant(variant))
}

func (h *Handler) UpdateVariant(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	variantID, ok := httpx.UUIDParam(c, "variantId")
	if !ok {
		return
	}
	var req dto.UpdateVariantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	variant, err := h.svc.UpdateVariant(c.Request.Context(), tenantID, productID, variantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromVariant(variant))
}

func (h *Handler) DeleteVariant(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	variantID, ok := httpx.UUIDParam(c, "variantId")
	if !ok {
		return
	}
	if err := h.svc.DeleteVariant(c.Request.Context(), tenantID, productID, variantID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ---- images (S3 direct upload) ----

// CreateImageUploadURL — POST .../images/upload-url  → presigned PUT URL.
func (h *Handler) CreateImageUploadURL(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.ImageUploadURLRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	ticket, err := h.svc.CreateImageUploadURL(c.Request.Context(), tenantID, productID, req.ContentType)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.ImageUploadURLResponse{
		ImageID:    ticket.ImageID,
		StorageKey: ticket.StorageKey,
		UploadURL:  ticket.UploadURL,
		PublicURL:  ticket.PublicURL,
		ExpiresIn:  ticket.ExpiresIn,
	})
}

// ConfirmImage — POST .../images  → register an uploaded object.
func (h *Handler) ConfirmImage(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	var req dto.ConfirmImageRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	img, err := h.svc.ConfirmImage(c.Request.Context(), tenantID, productID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromImage(img))
}

func (h *Handler) UpdateImage(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	imageID, ok := httpx.UUIDParam(c, "imageId")
	if !ok {
		return
	}
	var req dto.UpdateImageRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	img, err := h.svc.UpdateImage(c.Request.Context(), tenantID, productID, imageID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromImage(img))
}

func (h *Handler) DeleteImage(c *gin.Context) {
	tenantID, productID, ok := h.productScope(c)
	if !ok {
		return
	}
	imageID, ok := httpx.UUIDParam(c, "imageId")
	if !ok {
		return
	}
	if err := h.svc.DeleteImage(c.Request.Context(), tenantID, productID, imageID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ---- helpers ----

func (h *Handler) productScope(c *gin.Context) (tenantID, productID uuid.UUID, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	productID, ok = httpx.UUIDParam(c, "productId")
	return
}
