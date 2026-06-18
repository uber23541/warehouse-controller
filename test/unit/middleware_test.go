package unit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"warehouse-controller/internal/middleware"
	middlewaremock "warehouse-controller/internal/mocks/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func protectedEngine(v middleware.AccessValidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(middleware.AuthRequired(v))
	e.GET("/protected", func(c *gin.Context) {
		sid, _ := c.Get(middleware.ContextSessionKey)
		c.JSON(http.StatusOK, gin.H{"session_id": sid})
	})
	return e
}

// setup == nil → ValidateAccess не должен вызываться (запрос отсекается раньше).
func TestAuthRequired(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		setup    func(*middlewaremock.MockAccessValidator)
		wantCode int
		wantBody string
	}{
		{
			name: "missing header", header: "",
			wantCode: http.StatusUnauthorized, wantBody: "missing bearer token",
		},
		{
			name: "non-bearer header", header: "Basic abc",
			wantCode: http.StatusUnauthorized, wantBody: "missing bearer token",
		},
		{
			name: "invalid token", header: "Bearer bad-token",
			setup: func(v *middlewaremock.MockAccessValidator) {
				v.EXPECT().ValidateAccess(mock.Anything, "bad-token").Return("", errors.New("rejected")).Once()
			},
			wantCode: http.StatusUnauthorized, wantBody: "invalid token",
		},
		{
			name: "valid token", header: "Bearer good-token",
			setup: func(v *middlewaremock.MockAccessValidator) {
				v.EXPECT().ValidateAccess(mock.Anything, "good-token").Return("session-42", nil).Once()
			},
			wantCode: http.StatusOK, wantBody: "session-42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := middlewaremock.NewMockAccessValidator(t)
			if tt.setup != nil {
				tt.setup(v)
			}
			e := protectedEngine(v)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantBody)
		})
	}
}
