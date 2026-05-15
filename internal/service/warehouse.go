package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"warehouse-controller/internal/cache"
	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/repo/dbmodel"

	"go.uber.org/zap"
)

const cacheTTL = 10 * time.Second

type WarehouseService struct {
	repo  repo.ProductRepository
	cache cache.Cache
	log   *zap.Logger
}

func NewWarehouseService(repo repo.ProductRepository, c cache.Cache, log *zap.Logger) *WarehouseService {
	return &WarehouseService{repo: repo, cache: c, log: log}
}

func (s *WarehouseService) CreateProduct(ctx context.Context, req domain.CreateProductParams) (int64, error) {
	return s.repo.Create(ctx, &dbmodel.Product{
		ProductName:  req.ProductName,
		Manufacturer: req.Manufacturer,
		Category:     req.Category,
		Count:        req.Count,
		Price:        req.Price,
	})
}

func (s *WarehouseService) GetProductByID(ctx context.Context, req domain.GetProductParams) (*domain.Product, error) {
	key := fmt.Sprintf("product:%d", req.ID)

	data, err := s.cache.Get(ctx, key)
	if err == nil {
		var cached domain.Product
		if json.Unmarshal(data, &cached) == nil {
			s.log.Debug("cache hit", zap.Int64("id", req.ID))
			return &cached, nil
		}
	}

	dbProduct, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if dbProduct == nil {
		return nil, nil
	}

	domainProduct := repo.ToDomainProduct(dbProduct)
	data, err = json.Marshal(domainProduct)
	if err == nil {
		s.cache.Set(ctx, key, data, cacheTTL)
	}

	return domainProduct, nil
}

func (s *WarehouseService) DeleteProduct(ctx context.Context, req domain.DeleteProductParams) error {
	return s.repo.Delete(ctx, req.ID)
}

func (s *WarehouseService) RestoreProduct(ctx context.Context, req domain.RestoreProductParams) (*domain.Product, error) {
	dbProduct, err := s.repo.Restore(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if dbProduct == nil {
		return nil, nil
	}
	return repo.ToDomainProduct(dbProduct), nil
}

func (s *WarehouseService) SearchProducts(ctx context.Context, req domain.SearchProductsParams) ([]domain.Product, error) {
	key := fmt.Sprintf("products:search:%s", hashSearchParams(req))

	data, err := s.cache.Get(ctx, key)
	if err == nil {
		var cached []domain.Product
		if json.Unmarshal(data, &cached) == nil {
			s.log.Debug("cache hit search")
			return cached, nil
		}
	}

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

	domainProducts := repo.ToDomainProducts(dbProducts)
	cacheData, err := json.Marshal(domainProducts)
	if err == nil {
		s.cache.Set(ctx, key, cacheData, cacheTTL)
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
	return repo.ToDomainProduct(dbProduct), nil
}

func (s *WarehouseService) ListProducts(ctx context.Context, req domain.ListProductsParams) ([]domain.Product, error) {
	key := fmt.Sprintf("products:list:%d:%d", req.Limit, req.Offset)

	data, err := s.cache.Get(ctx, key)
	if err == nil {
		var cached []domain.Product
		if json.Unmarshal(data, &cached) == nil {
			s.log.Debug("cache hit list", zap.Int32("limit", req.Limit), zap.Int32("offset", req.Offset))
			return cached, nil
		}
	}

	dbProducts, err := s.repo.List(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	domainProducts := repo.ToDomainProducts(dbProducts)
	cacheData, err := json.Marshal(domainProducts)
	if err == nil {
		s.cache.Set(ctx, key, cacheData, cacheTTL)
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
