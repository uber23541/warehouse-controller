package sessioncache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"warehouse-controller/internal/cache"
)

type Store struct {
	cache cache.Cache
}

func New(c cache.Cache) *Store {
	return &Store{cache: c}
}

func refreshKey(sessionID string) string {
	return "refresh:" + sessionID
}

func accessKey(sessionID string) string {
	return "access:" + sessionID
}

func (s *Store) SetRefreshJTI(ctx context.Context, sessionID, jti string, ttl time.Duration) error {
	if err := s.cache.Set(ctx, refreshKey(sessionID), jti, ttl); err != nil {
		return fmt.Errorf("session set refresh: %w", err)
	}
	return nil
}

func (s *Store) GetRefreshJTI(ctx context.Context, sessionID string) (string, error) {
	return s.get(ctx, refreshKey(sessionID))
}

func (s *Store) SetAccessJTI(ctx context.Context, sessionID, jti string, ttl time.Duration) error {
	if err := s.cache.Set(ctx, accessKey(sessionID), jti, ttl); err != nil {
		return fmt.Errorf("session set access: %w", err)
	}
	return nil
}

func (s *Store) GetAccessJTI(ctx context.Context, sessionID string) (string, error) {
	return s.get(ctx, accessKey(sessionID))
}

func (s *Store) get(ctx context.Context, key string) (string, error) {
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return "", cache.ErrNotFound
		}
		return "", fmt.Errorf("session get: %w", err)
	}
	return string(data), nil
}
