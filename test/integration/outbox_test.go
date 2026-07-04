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
		repo.NewOutboxRepo(pool, 10),
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

func TestOutbox_DeadLetterAfterMaxAttempts(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t)

	const maxAttempts = 3
	r := repo.NewOutboxRepo(pool, maxAttempts)

	require.NoError(t, r.Save(ctx, outbox.Record{Topic: "t", Key: "poison", Payload: []byte(`{}`)}))
	require.NoError(t, r.Save(ctx, outbox.Record{Topic: "t", Key: "fresh", Payload: []byte(`{}`)}))

	msgs, err := r.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	poisonID := msgs[0].ID

	for i := 1; i <= maxAttempts; i++ {
		dead, err := r.MarkFailed(ctx, []int64{poisonID})
		require.NoError(t, err)
		if i < maxAttempts {
			assert.Zero(t, dead)
		} else {
			assert.EqualValues(t, 1, dead)
		}
	}

	msgs, err = r.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "fresh", msgs[0].Key)
}

func TestOutbox_FetchSkipsLockedRows(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t)

	r := repo.NewOutboxRepo(pool, 10)
	txm := postgres.NewTxManager(pool)

	require.NoError(t, r.Save(ctx, outbox.Record{Topic: "t", Key: "a", Payload: []byte(`{}`)}))
	require.NoError(t, r.Save(ctx, outbox.Record{Topic: "t", Key: "b", Payload: []byte(`{}`)}))

	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- txm.WithinTx(ctx, func(txCtx context.Context) error {
			msgs, err := r.FetchUnpublished(txCtx, 10)
			if err != nil {
				return err
			}
			ids := make([]int64, len(msgs))
			for i, m := range msgs {
				ids[i] = m.ID
			}
			close(locked)
			<-release
			return r.MarkPublished(txCtx, ids)
		})
	}()

	<-locked
	// Пока первая транзакция держит локи — конкурентная выборка пуста.
	msgs, err := r.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, msgs)

	close(release)
	require.NoError(t, <-done)

	// После коммита строки опубликованы — выбирать нечего.
	msgs, err = r.FetchUnpublished(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

// noopCache — заглушка кэша: CreateProduct его не использует.
type noopCache struct{}

func (noopCache) Get(context.Context, string) ([]byte, error)           { return nil, nil }
func (noopCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (noopCache) Delete(context.Context, string) error                  { return nil }
