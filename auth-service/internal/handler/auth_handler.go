package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/queue-booking/auth-service/internal/middleware"
	"github.com/queue-booking/auth-service/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Role     string `json:"role" binding:"required,oneof=customer provider"`
}

// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.auth.Register(req.Name, req.Email, req.Password, req.Role)
	if errors.Is(err, service.ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"role":       u.Role,
		"created_at": u.CreatedAt.UTC().Format(time.RFC3339),
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// spec only defines 200/401 for login
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email and password are required"})
		return
	}
	token, err := h.auth.Login(req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountSuspended) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": int(h.auth.TokenTTL().Seconds()),
	})
}

// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	jti, exp := middleware.TokenInfo(c)
	h.auth.Logout(jti, exp)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
