package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/auth-service/internal/middleware"
	"github.com/queue-booking/auth-service/internal/model"
	"github.com/queue-booking/auth-service/internal/service"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

// GET /api/users/me
func (h *UserHandler) GetMe(c *gin.Context) {
	u := middleware.CurrentUser(c)
	c.JSON(http.StatusOK, gin.H{
		"id":    u.ID,
		"name":  u.Name,
		"email": u.Email,
		"role":  u.Role,
	})
}

type updateMeRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=100"`
	Phone *string `json:"phone" binding:"omitempty,max=20"`
}

// PUT /api/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == nil && req.Phone == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of name or phone is required"})
		return
	}
	u, err := h.users.UpdateProfile(middleware.CurrentUser(c), req.Name, req.Phone)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "name": u.Name, "phone": u.Phone})
}

type listUsersQuery struct {
	Role  string `form:"role" binding:"omitempty,oneof=customer provider admin"`
	Page  int    `form:"page,default=1" binding:"min=1"`
	Limit int    `form:"limit,default=20" binding:"min=1,max=100"`
}

// GET /api/users (admin)
func (h *UserHandler) List(c *gin.Context) {
	var q listUsersQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	users, total, err := h.users.List(q.Role, q.Page, q.Limit)
	if err != nil {
		internalError(c, err)
		return
	}
	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		items = append(items, gin.H{"id": u.ID, "name": u.Name, "role": u.Role, "status": u.Status})
	}
	c.JSON(http.StatusOK, gin.H{"total": total, "users": items})
}

type setStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active suspended"`
}

// PUT /api/users/:id/status (admin)
func (h *UserHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var req setStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if uint(id) == middleware.CurrentUser(c).ID && req.Status == model.StatusSuspended {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot suspend your own account"})
		return
	}
	u, err := h.users.SetStatus(uint(id), req.Status)
	if errors.Is(err, service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "status": u.Status})
}

func internalError(c *gin.Context, err error) {
	log.Printf("internal error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
