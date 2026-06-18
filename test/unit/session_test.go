package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"warehouse-controller/internal/cache"
	sessioncache "warehouse-controller/internal/cache/session"
	cachemock "warehouse-controller/internal/mocks/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStore_SetAndGetAccessJTI(t *testing.T) {
	c := cachemock.NewMockCache(t)
	store := sessioncache.New(c)
	ctx := context.Background()
	ttl := time.Minute

	c.EXPECT().Set(ctx, "access:sess-1", "jti-1", ttl).Return(nil).Once()
	require.NoError(t, store.SetAccessJTI(ctx, "sess-1", "jti-1", ttl))

	c.EXPECT().Get(ctx, "access:sess-1").Return([]byte("jti-1"), nil).Once()
	got, err := store.GetAccessJTI(ctx, "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "jti-1", got)
}

func TestStore_SetAndGetRefreshJTI(t *testing.T) {
	c := cachemock.NewMockCache(t)
	store := sessioncache.New(c)
	ctx := context.Background()
	ttl := time.Hour

	c.EXPECT().Set(ctx, "refresh:sess-2", "jti-2", ttl).Return(nil).Once()
	require.NoError(t, store.SetRefreshJTI(ctx, "sess-2", "jti-2", ttl))

	c.EXPECT().Get(ctx, "refresh:sess-2").Return([]byte("jti-2"), nil).Once()
	got, err := store.GetRefreshJTI(ctx, "sess-2")
	require.NoError(t, err)
	assert.Equal(t, "jti-2", got)
}

func TestStore_Get_NotFoundPropagated(t *testing.T) {
	c := cachemock.NewMockCache(t)
	store := sessioncache.New(c)
	ctx := context.Background()

	c.EXPECT().Get(ctx, "access:missing").Return(nil, cache.ErrNotFound).Once()

	_, err := store.GetAccessJTI(ctx, "missing")
	assert.ErrorIs(t, err, cache.ErrNotFound)
}

func TestStore_Get_OtherErrorWrapped(t *testing.T) {
	c := cachemock.NewMockCache(t)
	store := sessioncache.New(c)
	ctx := context.Background()
	boom := errors.New("connection refused")

	c.EXPECT().Get(ctx, "refresh:sess").Return(nil, boom).Once()

	_, err := store.GetRefreshJTI(ctx, "sess")
	require.Error(t, err)
	assert.NotErrorIs(t, err, cache.ErrNotFound)
	assert.ErrorIs(t, err, boom)
}

func TestStore_Set_ErrorWrapped(t *testing.T) {
	c := cachemock.NewMockCache(t)
	store := sessioncache.New(c)
	ctx := context.Background()
	boom := errors.New("write failed")

	c.EXPECT().Set(ctx, mock.Anything, mock.Anything, mock.Anything).Return(boom).Once()

	err := store.SetAccessJTI(ctx, "s", "j", time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}
