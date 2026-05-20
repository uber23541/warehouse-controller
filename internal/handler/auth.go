package handler

import (
	"errors"
	"net/http"

	"warehouse-controller/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	svc    *service.AuthService
	logger *zap.Logger
}

func NewAuthHandler(svc *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

func (h *AuthHandler) Token(c *gin.Context) {
	pair, err := h.svc.IssuePair(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to issue token pair", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}
	c.JSON(http.StatusOK, pair)
}

type refreshRequest struct {
	Refresh string `json:"refresh" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.Refresh)
	if err != nil {
		if errors.Is(err, service.ErrRefreshRejected) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		h.logger.Error("failed to refresh tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh tokens"})
		return
	}
	c.JSON(http.StatusOK, pair)
}
