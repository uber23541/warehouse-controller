package middleware

import (
	"net/http"
	"strings"

	"warehouse-controller/internal/service"

	"github.com/gin-gonic/gin"
)

const ContextSessionKey = "session_id"

func AuthRequired(authSvc *service.AuthService) gin.HandlerFunc {
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
