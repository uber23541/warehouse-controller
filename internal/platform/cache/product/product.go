package productcache

import (
	"context"
	"encoding/json"
	"time"

	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/platform/cache"
)

type Product struct {
	ID           int64  `json:"id"`
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`
	Category     string `json:"category"`
	Price        int64  `json:"price"`
	Count        int32  `json:"count"`
}

func FromDomain(p *domain.Product) *Product {
	if p == nil {
		return nil
	}
	return &Product{
		ID:           p.ID,
		ProductName:  p.ProductName,
		Manufacturer: p.Manufacturer,
		Category:     p.Category,
		Price:        p.Price,
		Count:        p.Count,
	}
}

func FromDomainSlice(products []domain.Product) []Product {
	result := make([]Product, len(products))
	for i := range products {
		result[i] = *FromDomain(&products[i])
	}
	return result
}

func (p *Product) ToDomain() *domain.Product {
	if p == nil {
		return nil
	}
	return &domain.Product{
		ID:           p.ID,
		ProductName:  p.ProductName,
		Manufacturer: p.Manufacturer,
		Category:     p.Category,
		Price:        p.Price,
		Count:        p.Count,
	}
}

func ToDomainSlice(products []Product) []domain.Product {
	result := make([]domain.Product, len(products))
	for i := range products {
		result[i] = *products[i].ToDomain()
	}
	return result
}

func SetProduct(ctx context.Context, c cache.Cache, key string, p *Product, ttl time.Duration) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

func GetProduct(ctx context.Context, c cache.Cache, key string) (*Product, error) {
	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var p Product
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func SetProducts(ctx context.Context, c cache.Cache, key string, products []Product, ttl time.Duration) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

func GetProducts(ctx context.Context, c cache.Cache, key string) ([]Product, error) {
	data, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var products []Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, err
	}
	return products, nil
}
