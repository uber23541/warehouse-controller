//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/cache"
	sessioncache "warehouse-controller/internal/cache/session"
	"warehouse-controller/internal/event"
	"warehouse-controller/internal/handler"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func buildEngine(t *testing.T) *gin.Engine {
	t.Helper()
	log := zap.NewNop()

	productRepo := repo.NewProductRepo(startPostgres(t))
	redisCache := cache.New(startRedis(t))

	warehouseSvc := service.NewWarehouseService(productRepo, redisCache, event.NewNoopPublisher(), log)
	issuer := auth.NewIssuer("e2e-secret", 15*time.Minute, time.Hour)
	authSvc := service.NewAuthService(issuer, sessioncache.New(redisCache))

	warehouseH := handler.NewWarehouseHandler(warehouseSvc, log)
	authH := handler.NewAuthHandler(authSvc, log)

	return handler.NewRouter(warehouseH, authH, authSvc, log).Engine()
}

func request(t *testing.T, e *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, r)
	return rec
}

func TestE2E_FullProductLifecycle(t *testing.T) {
	e := buildEngine(t)

	tokenRec := request(t, e, http.MethodPost, "/auth/token", "", "")
	require.Equal(t, http.StatusOK, tokenRec.Code)
	var pair service.TokenPair
	require.NoError(t, json.Unmarshal(tokenRec.Body.Bytes(), &pair))
	require.NotEmpty(t, pair.Access)
	access := pair.Access

	createRec := request(t, e, http.MethodPost, "/products", access, `{"product_name":"Молоток","manufacturer":"Зубр","category":"Инструменты","price":1299,"count":5}`)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	require.Positive(t, created.ID)
	id := strconv.FormatInt(created.ID, 10)

	getRec := request(t, e, http.MethodGet, "/products/"+id, access, "")
	require.Equal(t, http.StatusOK, getRec.Code)
	assert.Contains(t, getRec.Body.String(), "Молоток")

	patchRec := request(t, e, http.MethodPatch, "/products/"+id, access, `{"price":2000}`)
	require.Equal(t, http.StatusOK, patchRec.Code)
	assert.Contains(t, patchRec.Body.String(), "2000")

	delRec := request(t, e, http.MethodDelete, "/products/"+id, access, "")
	require.Equal(t, http.StatusNoContent, delRec.Code)

	restoreRec := request(t, e, http.MethodPut, "/products/"+id+"/restore", access, "")
	require.Equal(t, http.StatusOK, restoreRec.Code)
}

func TestE2E_AuthGuards(t *testing.T) {
	e := buildEngine(t)

	noToken := request(t, e, http.MethodGet, "/products", "", "")
	assert.Equal(t, http.StatusUnauthorized, noToken.Code)

	badToken := request(t, e, http.MethodGet, "/products", "garbage", "")
	assert.Equal(t, http.StatusUnauthorized, badToken.Code)
}

func TestE2E_HealthAndMetrics(t *testing.T) {
	e := buildEngine(t)

	health := request(t, e, http.MethodGet, "/health", "", "")
	assert.Equal(t, http.StatusOK, health.Code)

	metrics := request(t, e, http.MethodGet, "/metrics", "", "")
	require.Equal(t, http.StatusOK, metrics.Code)
	assert.Contains(t, metrics.Body.String(), "http_requests_total")
}
