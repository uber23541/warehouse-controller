package handler

import (
	"net/http"
	"strings"

	"warehouse-controller/internal/auth"

	"github.com/gin-gonic/gin"
)

const ContextSessionKey = "session_id"

func AuthRequired(issuer *auth.Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(header, prefix)

		claims, err := issuer.ParseAccess(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(ContextSessionKey, claims.Subject)
		c.Next()
	}
}
