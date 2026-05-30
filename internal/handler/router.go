package handler

import (
	"net/http"
	"time"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/middleware"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type Router struct {
	warehouse *WarehouseHandler
	auth      *AuthHandler
	issuer    *auth.Issuer
	logger    *zap.Logger
}

func NewRouter(warehouseH *WarehouseHandler, authH *AuthHandler, issuer *auth.Issuer, logger *zap.Logger) *Router {
	return &Router{
		warehouse: warehouseH,
		auth:      authH,
		issuer:    issuer,
		logger:    logger,
	}
}

func (r *Router) Engine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(ginzap.Ginzap(r.logger, time.RFC3339, true))
	e.Use(ginzap.RecoveryWithZap(r.logger, true))
	e.Use(middleware.Metrics())

	r.registerRoutes(e)
	return e
}

func (r *Router) registerRoutes(e *gin.Engine) {
	e.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	e.GET("/metrics", gin.WrapH(promhttp.Handler()))
	e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	e.POST("/auth/token", r.auth.Token)
	e.POST("/auth/refresh", r.auth.Refresh)

	protected := e.Group("/", middleware.AuthRequired(r.issuer))
	protected.POST("/products", r.warehouse.CreateProduct)
	protected.GET("/products/:id", r.warehouse.GetProductByID)
	protected.DELETE("/products/:id", r.warehouse.DeleteProduct)
	protected.PUT("/products/:id/restore", r.warehouse.RestoreProduct)
	protected.GET("/products/search", r.warehouse.SearchProducts)
	protected.PATCH("/products/:id", r.warehouse.PatchProducts)
	protected.GET("/products", r.warehouse.ListProducts)
}
