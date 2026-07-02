package handler

import (
	"context"
	"errors"
	"net/http"

	"warehouse-controller/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthService interface {
	IssuePair(ctx context.Context) (service.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (service.TokenPair, error)
}

type AuthHandler struct {
	svc    AuthService
	logger *zap.Logger
}

func NewAuthHandler(svc AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// Token godoc
// @Summary      Выдать пару токенов
// @Description  Создаёт новую пару access/refresh токенов без проверки учётных данных
// @Tags         auth
// @Produce      json
// @Success      200  {object}  handler.TokenPairResponse
// @Failure      500  {object}  handler.ErrorResponse
// @Router       /auth/token [post]
func (h *AuthHandler) Token(c *gin.Context) {
	pair, err := h.svc.IssuePair(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to issue token pair", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue tokens"})
		return
	}
	c.JSON(http.StatusOK, newTokenPairResponse(pair))
}

// Refresh godoc
// @Summary      Обновить пару токенов
// @Description  Принимает действующий refresh-токен и возвращает новую пару
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      handler.RefreshRequest  true  "Refresh-токен"
// @Success      200      {object}  handler.TokenPairResponse
// @Failure      400      {object}  handler.ErrorResponse
// @Failure      401      {object}  handler.ErrorResponse
// @Failure      500      {object}  handler.ErrorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
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
	c.JSON(http.StatusOK, newTokenPairResponse(pair))
}
