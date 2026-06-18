//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"warehouse-controller/internal/domain"
	"warehouse-controller/internal/outbox"
	"warehouse-controller/internal/platform/postgres"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// CreateProduct в одной транзакции должен записать и продукт, и событие в outbox.
func TestOutbox_ProductCreatedWritten(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t)

	svc := service.NewWarehouseService(
		repo.NewProductRepo(pool),
		noopCache{},
		postgres.NewTxManager(pool),
		outbox.NewStore(pool),
		zap.NewNop(),
	)

	id, err := svc.CreateProduct(ctx, domain.CreateProductParams{
		ProductName: "Дрель", Manufacturer: "Bosch", Category: "Инструменты", Price: 5000, Count: 3,
	})
	require.NoError(t, err)
	require.Positive(t, id)

	var (
		topic, key string
		payload    []byte
	)
	err = pool.QueryRow(ctx, `SELECT topic, key, payload FROM outbox WHERE published_at IS NULL ORDER BY id DESC LIMIT 1`).
		Scan(&topic, &key, &payload)
	require.NoError(t, err)

	assert.Equal(t, "product.created", topic)

	var ev struct {
		ID          int64  `json:"id"`
		ProductName string `json:"product_name"`
	}
	require.NoError(t, json.Unmarshal(payload, &ev))
	assert.Equal(t, id, ev.ID)
	assert.Equal(t, "Дрель", ev.ProductName)
}

// noopCache — заглушка кэша: CreateProduct его не использует.
type noopCache struct{}

func (noopCache) Get(context.Context, string) ([]byte, error)           { return nil, nil }
func (noopCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (noopCache) Delete(context.Context, string) error                  { return nil }
