package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"warehouse-controller/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	cacheTTL         = 10 * time.Second
	keyProduct       = "product:%d"
	keyProductList   = "products:list:%d:%d"
	keyProductSearch = "products:search:%s"
)

type WarehouseService struct {
	pool   *pgxpool.Pool
	cache  *redis.Client
	logger *zap.Logger
}

func NewWarehouseService(pool *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *WarehouseService {
	return &WarehouseService{pool: pool, cache: cache, logger: logger}
}

func (s *WarehouseService) CreateProduct(ctx context.Context, req domain.CreateProductParams) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (product_name, manufacturer, category, count, price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, req.ProductName, req.Manufacturer, req.Category, req.Count, req.Price).Scan(&id)
	return id, err
}

func (s *WarehouseService) GetProductByID(ctx context.Context, req domain.GetProductParams) (*domain.Product, error) {
	cacheKey := fmt.Sprintf(keyProduct, req.ID)

	data, err := s.cache.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var p domain.Product
		if json.Unmarshal(data, &p) == nil {
			s.logger.Debug("cache hit", zap.Int64("id", req.ID))
			return &p, nil
		}
	}

	row := s.pool.QueryRow(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE id = $1 AND deleted_at IS NULL
	`, req.ID)

	var p domain.Product
	err = row.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if data, err := json.Marshal(p); err == nil {
		s.cache.Set(ctx, cacheKey, data, cacheTTL)
	}

	return &p, nil
}

func (s *WarehouseService) DeleteProduct(ctx context.Context, req domain.DeleteProductParams) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE products
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, req.ID)
	return err
}

func (s *WarehouseService) RestoreProduct(ctx context.Context, req domain.RestoreProductParams) (*domain.Product, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE products
		SET deleted_at = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NOT NULL
		RETURNING id, product_name, manufacturer, category, count, price
	`, req.ID)

	var p domain.Product
	err := row.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (s *WarehouseService) SearchProducts(ctx context.Context, req domain.SearchProductsParams) ([]domain.Product, error) {
	cacheKey := fmt.Sprintf(keyProductSearch, hashSearchParams(req))

	data, err := s.cache.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var products []domain.Product
		if json.Unmarshal(data, &products) == nil {
			s.logger.Debug("cache hit search")
			return products, nil
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE deleted_at IS NULL
		  AND ($1::text IS NULL OR product_name ILIKE '%' || $1::text || '%')
		  AND ($2::text IS NULL OR manufacturer = $2::text)
		  AND ($3::text IS NULL OR category = $3::text)
		  AND ($4::bigint IS NULL OR price >= $4::bigint)
		  AND ($5::bigint IS NULL OR price <= $5::bigint)
		ORDER BY id DESC
		LIMIT $6 OFFSET $7
	`, req.ProductName, req.Manufacturer, req.Category, req.MinPrice, req.MaxPrice, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if data, err := json.Marshal(products); err == nil {
		s.cache.Set(ctx, cacheKey, data, cacheTTL)
	}

	return products, rows.Err()
}

func (s *WarehouseService) PatchProduct(ctx context.Context, req domain.PatchProductParams) (*domain.Product, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE products
		SET
			product_name = COALESCE($1::text, product_name),
			manufacturer = COALESCE($2::text, manufacturer),
			category = COALESCE($3::text, category),
			count = COALESCE($4::integer, count),
			price = COALESCE($5::bigint, price),
			updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING id, product_name, manufacturer, category, count, price
	`, req.ProductName, req.Manufacturer, req.Category, req.Count, req.Price, req.ID)

	var p domain.Product
	err := row.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (s *WarehouseService) ListProducts(ctx context.Context, req domain.ListProductsParams) ([]domain.Product, error) {
	cacheKey := fmt.Sprintf(keyProductList, req.Limit, req.Offset)

	data, err := s.cache.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var products []domain.Product
		if json.Unmarshal(data, &products) == nil {
			s.logger.Debug("cache hit list", zap.Int32("limit", req.Limit), zap.Int32("offset", req.Offset))
			return products, nil
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE deleted_at IS NULL
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	if data, err := json.Marshal(products); err == nil {
		s.cache.Set(ctx, cacheKey, data, cacheTTL)
	}

	return products, rows.Err()
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
