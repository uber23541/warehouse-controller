package handler

import (
	"context"

	"warehouse-controller/internal/domain"

	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type WarehouseService interface {
	CreateProduct(ctx context.Context, req domain.CreateProductParams) (int64, error)
	GetProductByID(ctx context.Context, req domain.GetProductParams) (*domain.Product, error)
	DeleteProduct(ctx context.Context, req domain.DeleteProductParams) error
	RestoreProduct(ctx context.Context, req domain.RestoreProductParams) (*domain.Product, error)
	SearchProducts(ctx context.Context, req domain.SearchProductsParams) ([]domain.Product, error)
	PatchProduct(ctx context.Context, req domain.PatchProductParams) (*domain.Product, error)
	ListProducts(ctx context.Context, req domain.ListProductsParams) ([]domain.Product, error)
}

type WarehouseHandler struct {
	svc    WarehouseService
	logger *zap.Logger
}

func NewWarehouseHandler(svc WarehouseService, logger *zap.Logger) *WarehouseHandler {
	return &WarehouseHandler{svc: svc, logger: logger}
}

// CreateProduct godoc
// @Summary      Создать продукт
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        request  body      handler.CreateProductRequest  true  "Параметры нового продукта"
// @Success      201      {object}  handler.CreateProductResponse
// @Failure      400      {object}  handler.ErrorResponse
// @Failure      500      {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products [post]
func (h *WarehouseHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to bind request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	id, err := h.svc.CreateProduct(c.Request.Context(), req.toDomain())
	if err != nil {
		h.logger.Error("failed to create product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, CreateProductResponse{ID: id})
}

// GetProductByID godoc
// @Summary      Получить продукт по ID
// @Tags         products
// @Produce      json
// @Param        id   path      int  true  "ID продукта"
// @Success      200  {object}  handler.ProductResponse
// @Failure      400  {object}  handler.ErrorResponse
// @Failure      500  {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products/{id} [get]
func (h *WarehouseHandler) GetProductByID(c *gin.Context) {
	var uri ProductIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	product, err := h.svc.GetProductByID(c.Request.Context(), domain.GetProductParams{ID: uri.ID})
	if err != nil {
		h.logger.Error("failed to get product by ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get product"})
		return
	}

	h.respondProduct(c, product)
}

// DeleteProduct godoc
// @Summary      Удалить продукт (soft delete)
// @Tags         products
// @Param        id   path  int  true  "ID продукта"
// @Success      204
// @Failure      400  {object}  handler.ErrorResponse
// @Failure      500  {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products/{id} [delete]
func (h *WarehouseHandler) DeleteProduct(c *gin.Context) {
	var uri ProductIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	if err := h.svc.DeleteProduct(c.Request.Context(), domain.DeleteProductParams{ID: uri.ID}); err != nil {
		h.logger.Error("failed to delete product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}

	c.Status(http.StatusNoContent)
}

// RestoreProduct godoc
// @Summary      Восстановить ранее удалённый продукт
// @Tags         products
// @Produce      json
// @Param        id   path      int  true  "ID продукта"
// @Success      200  {object}  handler.ProductResponse
// @Failure      400  {object}  handler.ErrorResponse
// @Failure      500  {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products/{id}/restore [put]
func (h *WarehouseHandler) RestoreProduct(c *gin.Context) {
	var uri ProductIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	product, err := h.svc.RestoreProduct(c.Request.Context(), domain.RestoreProductParams{ID: uri.ID})
	if err != nil {
		h.logger.Error("failed to restore product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore product"})
		return
	}

	h.respondProduct(c, product)
}

// SearchProducts godoc
// @Summary      Поиск продуктов
// @Description  Фильтрация продуктов по имени, производителю, категории и диапазону цены
// @Tags         products
// @Produce      json
// @Param        product_name   query     string  false  "Подстрока в названии"
// @Param        manufacturer   query     string  false  "Производитель"
// @Param        category       query     string  false  "Категория"
// @Param        min_price      query     int     false  "Минимальная цена"
// @Param        max_price      query     int     false  "Максимальная цена"
// @Param        limit          query     int     false  "Лимит результатов"
// @Param        offset         query     int     false  "Смещение"
// @Success      200            {array}   handler.ProductResponse
// @Failure      400            {object}  handler.ErrorResponse
// @Failure      500            {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products/search [get]
func (h *WarehouseHandler) SearchProducts(c *gin.Context) {
	var req SearchProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("failed to bind query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	products, err := h.svc.SearchProducts(c.Request.Context(), req.toDomain())
	if err != nil {
		h.logger.Error("failed to search products", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search products"})
		return
	}

	c.JSON(http.StatusOK, newProductResponses(products))
}

// PatchProducts godoc
// @Summary      Частичное обновление продукта
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        id       path      int                            true  "ID продукта"
// @Param        request  body      handler.PatchProductRequest    true  "Поля для обновления"
// @Success      200      {object}  handler.ProductResponse
// @Failure      400      {object}  handler.ErrorResponse
// @Failure      500      {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products/{id} [patch]
func (h *WarehouseHandler) PatchProducts(c *gin.Context) {
	var uri ProductIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		h.logger.Error("failed to bind URI parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URI parameters"})
		return
	}

	var req PatchProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("failed to bind request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	product, err := h.svc.PatchProduct(c.Request.Context(), req.toDomain(uri.ID))
	if err != nil {
		h.logger.Error("failed to patch product", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch product"})
		return
	}

	h.respondProduct(c, product)
}

// ListProducts godoc
// @Summary      Список продуктов
// @Tags         products
// @Produce      json
// @Param        limit   query     int  false  "Лимит результатов"
// @Param        offset  query     int  false  "Смещение"
// @Success      200     {array}   handler.ProductResponse
// @Failure      400     {object}  handler.ErrorResponse
// @Failure      500     {object}  handler.ErrorResponse
// @Security     BearerAuth
// @Router       /products [get]
func (h *WarehouseHandler) ListProducts(c *gin.Context) {
	var req ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("failed to bind query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}

	products, err := h.svc.ListProducts(c.Request.Context(), req.toDomain())
	if err != nil {
		h.logger.Error("failed to list products", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	c.JSON(http.StatusOK, newProductResponses(products))
}

func (h *WarehouseHandler) respondProduct(c *gin.Context, product *domain.Product) {
	if product == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, newProductResponse(product))
}
