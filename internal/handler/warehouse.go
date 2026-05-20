package handler

import (
	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/service"

	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type WarehouseHandler struct {
	svc    *service.WarehouseService
	logger *zap.Logger
}

func NewWarehouseHandler(svc *service.WarehouseService, logger *zap.Logger) *WarehouseHandler {
	return &WarehouseHandler{svc: svc, logger: logger}
}

func (h *WarehouseHandler) CreateProduct(c *gin.Context) {
	var req domain.CreateProductParams
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to bind request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	id, err := h.svc.CreateProduct(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to create product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WarehouseHandler) GetProductByID(c *gin.Context) {
	var req domain.GetProductParams
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	product, err := h.svc.GetProductByID(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to get product by ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *WarehouseHandler) DeleteProduct(c *gin.Context) {
	var req domain.DeleteProductParams
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	if err := h.svc.DeleteProduct(c.Request.Context(), req); err != nil {
		h.logger.Error("failed to delete product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *WarehouseHandler) RestoreProduct(c *gin.Context) {
	var req domain.RestoreProductParams
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	product, err := h.svc.RestoreProduct(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to restore product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore product"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *WarehouseHandler) SearchProducts(c *gin.Context) {
	var req domain.SearchProductsParams
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("failed to bind query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	products, err := h.svc.SearchProducts(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to search products", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search products"})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *WarehouseHandler) PatchProducts(c *gin.Context) {
	var req domain.PatchProductParams
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to bind request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	product, err := h.svc.PatchProduct(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to patch product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch product"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *WarehouseHandler) ListProducts(c *gin.Context) {
	var req domain.ListProductsParams
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("failed to bind query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	products, err := h.svc.ListProducts(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to list products", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, products)
}
