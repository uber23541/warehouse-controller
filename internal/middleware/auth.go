package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextSessionKey = "session_id"

// AccessValidator проверяет access-токен и возвращает идентификатор сессии.
// Узкий интерфейс позволяет подменять auth-сервис в тестах; его удовлетворяет
// *service.AuthService.
type AccessValidator interface {
	ValidateAccess(ctx context.Context, accessToken string) (string, error)
}

func AuthRequired(authSvc AccessValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(header, prefix)

		sessionID, err := authSvc.ValidateAccess(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ContextSessionKey, sessionID)
		c.Next()
	}
}
