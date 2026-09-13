package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/services"
)

type AddressHandler struct {
	svc *services.AddressService
}

func NewAddressHandler(svc *services.AddressService) *AddressHandler {
	return &AddressHandler{svc: svc}
}

// userScope returns the :userId being addressed, but only if the caller is that
// user or a super-admin. Otherwise it reports a 403 and returns ok=false.
func (h *AddressHandler) userScope(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return uuid.Nil, false
	}
	principal, ok := httpx.Principal(c)
	if !ok {
		return uuid.Nil, false
	}
	if principal.UserID != userID && !principal.IsSuperAdmin {
		_ = c.Error(httpx.Forbidden("you can only manage your own addresses"))
		return uuid.Nil, false
	}
	return userID, true
}

func (h *AddressHandler) Create(c *gin.Context) {
	userID, ok := h.userScope(c)
	if !ok {
		return
	}
	var req dto.CreateAddressRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	addr, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromAddress(addr))
}

func (h *AddressHandler) List(c *gin.Context) {
	userID, ok := h.userScope(c)
	if !ok {
		return
	}
	addrs, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromAddresses(addrs))
}

func (h *AddressHandler) Get(c *gin.Context) {
	userID, ok := h.userScope(c)
	if !ok {
		return
	}
	addressID, ok := httpx.UUIDParam(c, "addressId")
	if !ok {
		return
	}
	addr, err := h.svc.Get(c.Request.Context(), userID, addressID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromAddress(addr))
}

func (h *AddressHandler) Update(c *gin.Context) {
	userID, ok := h.userScope(c)
	if !ok {
		return
	}
	addressID, ok := httpx.UUIDParam(c, "addressId")
	if !ok {
		return
	}
	var req dto.UpdateAddressRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	addr, err := h.svc.Update(c.Request.Context(), userID, addressID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromAddress(addr))
}

func (h *AddressHandler) Delete(c *gin.Context) {
	userID, ok := h.userScope(c)
	if !ok {
		return
	}
	addressID, ok := httpx.UUIDParam(c, "addressId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), userID, addressID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}
