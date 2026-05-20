package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type redisCache struct {
	client *redis.Client
}

func New(client *redis.Client) Cache {
	return &redisCache{client: client}
}

func (c *redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func SetProduct(ctx context.Context, c Cache, key string, p *Product, ttl time.Duration) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

func GetProduct(ctx context.Context, c Cache, key string) (*Product, error) {
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

func SetProducts(ctx context.Context, c Cache, key string, products []Product, ttl time.Duration) error {
	data, err := json.Marshal(products)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

func GetProducts(ctx context.Context, c Cache, key string) ([]Product, error) {
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
