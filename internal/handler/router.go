package handler

import (
	"net/http"
	"time"

	"warehouse-controller/internal/auth"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func BuildRouter(warehouseH *WarehouseHandler, authH *AuthHandler, issuer *auth.Issuer, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.POST("/auth/token", authH.Token)
	r.POST("/auth/refresh", authH.Refresh)

	protected := r.Group("/", AuthRequired(issuer))
	protected.POST("/products", warehouseH.CreateProduct)
	protected.GET("/products/:id", warehouseH.GetProductByID)
	protected.DELETE("/products/:id", warehouseH.DeleteProduct)
	protected.PUT("/products/:id/restore", warehouseH.RestoreProduct)
	protected.GET("/products/search", warehouseH.SearchProducts)
	protected.PATCH("/products/:id", warehouseH.PatchProducts)
	protected.GET("/products", warehouseH.ListProducts)

	return r
}
