// Package httphandler contains the Gin HTTP handlers for the user module.
// Handlers never format errors themselves: they call c.Error(err) and return,
// letting internal/httpx.ErrorHandler render the response.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/services"
)

type UserHandler struct {
	svc *services.UserService
}

func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(c *gin.Context) {
	limit, offset := httpx.Pagination(c)
	users, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromUsers(users), limit, offset)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromUser(user))
}

// GetInternal — GET /api/v1/internal/users/:userId (X-Internal-Key).
// Other services use it to resolve a user's name + email, e.g. mail-service
// sending transactional email off a Kafka event.
func (h *UserHandler) GetInternal(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromUser(user))
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	var req dto.UpdateUserRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	user, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromUser(user))
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}
