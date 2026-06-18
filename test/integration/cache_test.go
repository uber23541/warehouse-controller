//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"warehouse-controller/internal/platform/cache"
	productcache "warehouse-controller/internal/platform/cache/product"
	sessioncache "warehouse-controller/internal/platform/cache/session"
	"warehouse-controller/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_SetGetDelete(t *testing.T) {
	ctx := context.Background()
	c := cache.New(startRedis(t))

	require.NoError(t, c.Set(ctx, "k", []byte("v"), time.Minute))

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("v"), got)

	require.NoError(t, c.Delete(ctx, "k"))

	_, err = c.Get(ctx, "k")
	assert.ErrorIs(t, err, cache.ErrNotFound)
}

func TestCache_MissingKey(t *testing.T) {
	ctx := context.Background()
	c := cache.New(startRedis(t))

	_, err := c.Get(ctx, "missing")
	assert.ErrorIs(t, err, cache.ErrNotFound)
}

func TestProductCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.New(startRedis(t))

	want := &domain.Product{ID: 1, ProductName: "Молоток", Manufacturer: "Зубр", Category: "Инструменты", Price: 1299, Count: 5}
	require.NoError(t, productcache.SetProduct(ctx, c, "product:1", productcache.FromDomain(want), time.Minute))

	got, err := productcache.GetProduct(ctx, c, "product:1")
	require.NoError(t, err)
	assert.Equal(t, want, got.ToDomain())
}

func TestSessionStore_RoundTripAndOverwrite(t *testing.T) {
	ctx := context.Background()
	store := sessioncache.New(cache.New(startRedis(t)))

	require.NoError(t, store.SetAccessJTI(ctx, "sess", "jti-1", time.Minute))
	got, err := store.GetAccessJTI(ctx, "sess")
	require.NoError(t, err)
	assert.Equal(t, "jti-1", got)

	require.NoError(t, store.SetAccessJTI(ctx, "sess", "jti-2", time.Minute))
	got, err = store.GetAccessJTI(ctx, "sess")
	require.NoError(t, err)
	assert.Equal(t, "jti-2", got)
}
