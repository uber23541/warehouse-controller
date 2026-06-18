package unit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/handler"
	handlermock "warehouse-controller/internal/mocks/handler"
	"warehouse-controller/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func warehouseEngine(t *testing.T) (*gin.Engine, *handlermock.MockWarehouseService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := handlermock.NewMockWarehouseService(t)
	h := handler.NewWarehouseHandler(svc, zap.NewNop())
	e := gin.New()
	e.POST("/products", h.CreateProduct)
	e.GET("/products/:id", h.GetProductByID)
	e.DELETE("/products/:id", h.DeleteProduct)
	e.PUT("/products/:id/restore", h.RestoreProduct)
	e.PATCH("/products/:id", h.PatchProducts)
	e.GET("/products", h.ListProducts)
	e.GET("/products/search", h.SearchProducts)
	return e, svc
}

func authEngine(t *testing.T) (*gin.Engine, *handlermock.MockAuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := handlermock.NewMockAuthService(t)
	h := handler.NewAuthHandler(svc, zap.NewNop())
	e := gin.New()
	e.POST("/auth/token", h.Token)
	e.POST("/auth/refresh", h.Refresh)
	return e, svc
}

func httpDo(e *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, r)
	return rec
}

// setup == nil означает, что сервис не должен вызываться (мок без EXPECT это
// проверит) — так строки с биндинг-ошибками подтверждают код 400 без обращения
// к бизнес-логике.
func TestWarehouseHandlers(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		setup    func(*handlermock.MockWarehouseService)
		wantCode int
		wantBody string
	}{
		{
			name: "create: ok", method: http.MethodPost, path: "/products",
			body: `{"product_name":"Молоток","price":100}`,
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().CreateProduct(mock.Anything, mock.MatchedBy(func(p domain.CreateProductParams) bool {
					return p.ProductName == "Молоток"
				})).Return(int64(42), nil).Once()
			},
			wantCode: http.StatusCreated,
			wantBody: `"id":42`,
		},
		{
			name:   "create: invalid json",
			method: http.MethodPost, path: "/products",
			body:     `{not json`,
			wantCode: http.StatusBadRequest,
			wantBody: "invalid request body",
		},
		{
			name:   "create: service error",
			method: http.MethodPost,
			path:   "/products",
			body:   `{"product_name":"X"}`,
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().CreateProduct(mock.Anything, mock.Anything).Return(int64(0), errors.New("boom")).Once()
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "get: ok", method: http.MethodGet, path: "/products/7",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().GetProductByID(mock.Anything, domain.GetProductParams{ID: 7}).
					Return(&domain.Product{ID: 7, ProductName: "Дрель"}, nil).Once()
			},
			wantCode: http.StatusOK,
			wantBody: "Дрель",
		},
		{
			name:   "get: non-numeric id",
			method: http.MethodGet, path: "/products/abc",
			wantCode: http.StatusBadRequest,
			wantBody: "invalid URI parameters",
		},
		{
			name:   "get: service error",
			method: http.MethodGet, path: "/products/7",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().GetProductByID(mock.Anything, mock.Anything).Return(nil, errors.New("db")).Once()
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:   "delete: ok",
			method: http.MethodDelete, path: "/products/3",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().DeleteProduct(mock.Anything, domain.DeleteProductParams{ID: 3}).Return(nil).Once()
			},
			wantCode: http.StatusNoContent,
		},
		{
			name:   "delete: non-numeric id",
			method: http.MethodDelete, path: "/products/abc",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "restore: ok",
			method: http.MethodPut, path: "/products/4/restore",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().RestoreProduct(mock.Anything, domain.RestoreProductParams{ID: 4}).
					Return(&domain.Product{ID: 4}, nil).Once()
			},
			wantCode: http.StatusOK,
		},
		{
			name: "restore: non-numeric id", method: http.MethodPut, path: "/products/abc/restore",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "list: ok", method: http.MethodGet,
			path: "/products?limit=2&offset=0",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().ListProducts(mock.Anything, mock.Anything).
					Return([]domain.Product{{ID: 1}, {ID: 2}}, nil).Once()
			},
			wantCode: http.StatusOK,
			wantBody: `"id":1`,
		},
		{
			name:     "list: non-numeric limit",
			method:   http.MethodGet,
			path:     "/products?limit=abc",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "search: ok",
			method: http.MethodGet,
			path:   "/products/search?product_name=Мол",
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().SearchProducts(mock.Anything, mock.MatchedBy(func(p domain.SearchProductsParams) bool {
					return p.ProductName != nil && *p.ProductName == "Мол"
				})).Return([]domain.Product{{ID: 1, ProductName: "Молоток"}}, nil).Once()
			},
			wantCode: http.StatusOK,
			wantBody: "Молоток",
		},
		{
			name:     "search: non-numeric min_price",
			method:   http.MethodGet,
			path:     "/products/search?min_price=abc",
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "patch: ok",
			method: http.MethodPatch,
			path:   "/products/5",
			body:   `{"product_name":"New"}`,
			setup: func(s *handlermock.MockWarehouseService) {
				s.EXPECT().PatchProduct(mock.Anything, mock.MatchedBy(func(p domain.PatchProductParams) bool {
					return p.ID == 5 && p.ProductName != nil && *p.ProductName == "New"
				})).Return(&domain.Product{ID: 5, ProductName: "New"}, nil).Once()
			},
			wantCode: http.StatusOK,
			wantBody: "New",
		},
		{
			name:   "patch: non-numeric id",
			method: http.MethodPatch, path: "/products/abc",
			body:     `{"price":1}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "patch: invalid json", method: http.MethodPatch,
			path:     "/products/5",
			body:     `{bad`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, svc := warehouseEngine(t)
			if tt.setup != nil {
				tt.setup(svc)
			}

			rec := httpDo(e, tt.method, tt.path, tt.body)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody != "" {
				assert.Contains(t, rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestAuthHandlers(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		setup    func(*handlermock.MockAuthService)
		wantCode int
		wantBody string
	}{
		{
			name: "token: ok", method: http.MethodPost, path: "/auth/token",
			setup: func(s *handlermock.MockAuthService) {
				s.EXPECT().IssuePair(mock.Anything).Return(service.TokenPair{Access: "a", Refresh: "r"}, nil).Once()
			},
			wantCode: http.StatusOK, wantBody: `"access":"a"`,
		},
		{
			name: "token: service error", method: http.MethodPost, path: "/auth/token",
			setup: func(s *handlermock.MockAuthService) {
				s.EXPECT().IssuePair(mock.Anything).Return(service.TokenPair{}, errors.New("boom")).Once()
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "refresh: empty body", method: http.MethodPost, path: "/auth/refresh", body: `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "refresh: rejected", method: http.MethodPost, path: "/auth/refresh", body: `{"refresh":"tok"}`,
			setup: func(s *handlermock.MockAuthService) {
				s.EXPECT().Refresh(mock.Anything, "tok").Return(service.TokenPair{}, service.ErrRefreshRejected).Once()
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "refresh: ok", method: http.MethodPost, path: "/auth/refresh", body: `{"refresh":"tok"}`,
			setup: func(s *handlermock.MockAuthService) {
				s.EXPECT().Refresh(mock.Anything, "tok").Return(service.TokenPair{Access: "a2", Refresh: "r2"}, nil).Once()
			},
			wantCode: http.StatusOK, wantBody: "a2",
		},
		{
			name: "refresh: internal error", method: http.MethodPost, path: "/auth/refresh", body: `{"refresh":"tok"}`,
			setup: func(s *handlermock.MockAuthService) {
				s.EXPECT().Refresh(mock.Anything, "tok").Return(service.TokenPair{}, errors.New("redis down")).Once()
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, svc := authEngine(t)
			if tt.setup != nil {
				tt.setup(svc)
			}

			rec := httpDo(e, tt.method, tt.path, tt.body)

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody != "" {
				assert.Contains(t, rec.Body.String(), tt.wantBody)
			}
		})
	}
}
