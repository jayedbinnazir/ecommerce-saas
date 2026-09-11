// Package httphandler contains the Gin HTTP handlers for the stock module.
package httphandler

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/inventory-service/internal/httpx"
	"github.com/jayedbinnazir/inventory-service/internal/platform"
	"github.com/jayedbinnazir/inventory-service/internal/stock/domain"
	"github.com/jayedbinnazir/inventory-service/internal/stock/dto"
	"github.com/jayedbinnazir/inventory-service/internal/stock/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// ---------------------------------------------------------------------
// Storefront (public) — availability only, no internal counts
// ---------------------------------------------------------------------

func (h *Handler) Availability(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantID, sku)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.Availability(item))
}

// ---------------------------------------------------------------------
// Management (tenant ADMIN / MANAGER)
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
		LowOnly:  strings.EqualFold(c.Query("low_stock"), "true"),
		Page:     platform.Page{Limit: limit, Offset: offset},
	}

	items, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromStockItems(items), limit, offset)
}

func (h *Handler) Create(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreateStockItemRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromStockItem(item))
}

func (h *Handler) Get(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantID, sku)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromStockItem(item))
}

func (h *Handler) Update(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	var req dto.UpdateStockItemRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), tenantID, sku, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromStockItem(item))
}

func (h *Handler) Delete(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), tenantID, sku); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ---- operations ----

func (h *Handler) Receive(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	var req dto.ReceiveRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Receive(c.Request.Context(), tenantID, sku, h.actor(c), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromStockItem(item))
}

func (h *Handler) Adjust(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	var req dto.AdjustRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	item, err := h.svc.Adjust(c.Request.Context(), tenantID, sku, h.actor(c), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromStockItem(item))
}

func (h *Handler) Reserve(c *gin.Context) { h.move(c, h.svc.Reserve) }
func (h *Handler) Release(c *gin.Context) { h.move(c, h.svc.Release) }
func (h *Handler) Ship(c *gin.Context)    { h.move(c, h.svc.Ship) }

type moveFunc func(ctx context.Context, tenantID uuid.UUID, sku string, actor uuid.UUID, req dto.MoveRequest) (*domain.StockItem, error)

// ---- bulk operations (internal, service-to-service) ----

func (h *Handler) ReserveBulk(c *gin.Context) { h.bulk(c, h.svc.ReserveBulk) }
func (h *Handler) ReleaseBulk(c *gin.Context) { h.bulk(c, h.svc.ReleaseBulk) }
func (h *Handler) ShipBulk(c *gin.Context)    { h.bulk(c, h.svc.ShipBulk) }
func (h *Handler) RestockBulk(c *gin.Context) { h.bulk(c, h.svc.RestockBulk) }

type bulkFunc func(ctx context.Context, tenantID, actor uuid.UUID, req dto.BulkMoveRequest) ([]domain.StockItem, error)

func (h *Handler) bulk(c *gin.Context, fn bulkFunc) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.BulkMoveRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	items, err := fn(c.Request.Context(), tenantID, h.actor(c), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.BulkMoveResponse{Items: dto.FromStockItems(items)})
}

func (h *Handler) Movements(c *gin.Context) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)
	items, err := h.svc.Movements(c.Request.Context(), tenantID, sku, platform.Page{Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromMovements(items), limit, offset)
}

// ---- helpers ----

func (h *Handler) move(c *gin.Context, fn moveFunc) {
	tenantID, sku, ok := h.scope(c)
	if !ok {
		return
	}
	var req dto.MoveRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	item, err := fn(c.Request.Context(), tenantID, sku, h.actor(c), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromStockItem(item))
}

// scope reads :tenantId (a UUID) and :sku (a non-empty string) from the path.
func (h *Handler) scope(c *gin.Context) (tenantID uuid.UUID, sku string, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	sku = strings.TrimSpace(c.Param("sku"))
	if sku == "" {
		_ = c.Error(httpx.BadRequest("path parameter sku is required"))
		return tenantID, "", false
	}
	return tenantID, sku, true
}

// actor is the authenticated user, or uuid.Nil on the public route.
func (h *Handler) actor(c *gin.Context) uuid.UUID {
	if p, ok := platform.PrincipalFromContext(c.Request.Context()); ok {
		return p.UserID
	}
	return uuid.Nil
}
