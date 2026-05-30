package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"warehouse-controller/internal/cache"
	productcache "warehouse-controller/internal/cache/product"
	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/event"
	"warehouse-controller/internal/metrics"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/repo/dbmodel"

	"go.uber.org/zap"
)

const cacheTTL = 10 * time.Second

type WarehouseService struct {
	repo   repo.ProductRepository
	cache  cache.Cache
	events event.Publisher
	log    *zap.Logger
}

func NewWarehouseService(r repo.ProductRepository, c cache.Cache, events event.Publisher, log *zap.Logger) *WarehouseService {
	return &WarehouseService{repo: r, cache: c, events: events, log: log}
}

func (s *WarehouseService) CreateProduct(ctx context.Context, req domain.CreateProductParams) (int64, error) {
	id, err := s.repo.Create(ctx, &dbmodel.Product{
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		Count:        req.Count,
		Price:        req.Price,
	})
	if err != nil {
		return 0, err
	}

	if err := s.events.Publish(ctx, event.ProductCreated{
		ID:           id,
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		Price:        req.Price,
		Count:        req.Count,
	}); err != nil {
		s.log.Warn("publish product.created failed", zap.Int64("id", id), zap.Error(err))
	}

	return id, nil
}

func (s *WarehouseService) GetProductByID(ctx context.Context, req domain.GetProductParams) (*domain.Product, error) {
	key := fmt.Sprintf("product:%d", req.ID)

	if cached, err := productcache.GetProduct(ctx, s.cache, key); err == nil {
		metrics.CacheHit("product")
		s.log.Debug("cache hit", zap.Int64("id", req.ID))
		return cached.ToDomain(), nil
	}
	metrics.CacheMiss("product")

	dbProduct, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if dbProduct == nil {
		return nil, nil
	}

	domainProduct := dbProduct.ToDomain()
	if err := productcache.SetProduct(ctx, s.cache, key, productcache.FromDomain(domainProduct), cacheTTL); err != nil {
		s.log.Warn("cache set failed", zap.Error(err))
	}

	return domainProduct, nil
}

func (s *WarehouseService) DeleteProduct(ctx context.Context, req domain.DeleteProductParams) error {
	if err := s.repo.Delete(ctx, req.ID); err != nil {
		return err
	}

	if err := s.events.Publish(ctx, event.ProductDeleted{ID: req.ID}); err != nil {
		s.log.Warn("publish product.deleted failed", zap.Int64("id", req.ID), zap.Error(err))
	}

	return nil
}

func (s *WarehouseService) RestoreProduct(ctx context.Context, req domain.RestoreProductParams) (*domain.Product, error) {
	dbProduct, err := s.repo.Restore(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if dbProduct == nil {
		return nil, nil
	}
	return dbProduct.ToDomain(), nil
}

func (s *WarehouseService) SearchProducts(ctx context.Context, req domain.SearchProductsParams) ([]domain.Product, error) {
	key := fmt.Sprintf("products:search:%s", hashSearchParams(req))

	if cached, err := productcache.GetProducts(ctx, s.cache, key); err == nil {
		metrics.CacheHit("search")
		s.log.Debug("cache hit search")
		return productcache.ToDomainSlice(cached), nil
	}
	metrics.CacheMiss("search")

	dbProducts, err := s.repo.Search(ctx, dbmodel.ProductFilter{
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		MinPrice:     req.MinPrice,
		MaxPrice:     req.MaxPrice,
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
	if err != nil {
		return nil, err
	}

	domainProducts := dbmodel.ToDomainSlice(dbProducts)
	if err := productcache.SetProducts(ctx, s.cache, key, productcache.FromDomainSlice(domainProducts), cacheTTL); err != nil {
		s.log.Warn("cache set failed", zap.Error(err))
	}

	return domainProducts, nil
}

func (s *WarehouseService) PatchProduct(ctx context.Context, req domain.PatchProductParams) (*domain.Product, error) {
	dbProduct, err := s.repo.Patch(ctx, req.ID, dbmodel.ProductPatch{
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		Count:        req.Count,
		Price:        req.Price,
	})
	if err != nil {
		return nil, err
	}
	if dbProduct == nil {
		return nil, nil
	}
	return dbProduct.ToDomain(), nil
}

func (s *WarehouseService) ListProducts(ctx context.Context, req domain.ListProductsParams) ([]domain.Product, error) {
	key := fmt.Sprintf("products:list:%d:%d", req.Limit, req.Offset)

	if cached, err := productcache.GetProducts(ctx, s.cache, key); err == nil {
		metrics.CacheHit("list")
		s.log.Debug("cache hit list", zap.Int32("limit", req.Limit), zap.Int32("offset", req.Offset))
		return productcache.ToDomainSlice(cached), nil
	}
	metrics.CacheMiss("list")

	dbProducts, err := s.repo.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	domainProducts := dbmodel.ToDomainSlice(dbProducts)
	if err := productcache.SetProducts(ctx, s.cache, key, productcache.FromDomainSlice(domainProducts), cacheTTL); err != nil {
		s.log.Warn("cache set failed", zap.Error(err))
	}

	return domainProducts, nil
}

func hashSearchParams(req domain.SearchProductsParams) string {
	var parts []string

	if req.ProductName != nil {
		parts = append(parts, *req.ProductName)
	}
	if req.Manufacturer != nil {
		parts = append(parts, *req.Manufacturer)
	}
	if req.Category != nil {
		parts = append(parts, *req.Category)
	}
	if req.MinPrice != nil {
		parts = append(parts, fmt.Sprintf("%d", *req.MinPrice))
	}
	if req.MaxPrice != nil {
		parts = append(parts, fmt.Sprintf("%d", *req.MaxPrice))
	}
	parts = append(parts, fmt.Sprintf("%d", req.Limit))
	parts = append(parts, fmt.Sprintf("%d", req.Offset))

	raw := strings.Join(parts, "|")
	//sum := sha256.Sum256([]byte(raw))

	//return hex.EncodeToString(sum[:])
	return raw
}
