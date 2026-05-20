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

func sessionKey(sessionID string) string {
	return "refresh:" + sessionID
}

func (s *Store) SetRefreshJTI(ctx context.Context, sessionID, jti string, ttl time.Duration) error {
	if err := s.cache.Set(ctx, sessionKey(sessionID), jti, ttl); err != nil {
		return fmt.Errorf("session set: %w", err)
	}
	return nil
}

func (s *Store) GetRefreshJTI(ctx context.Context, sessionID string) (string, error) {
	data, err := s.cache.Get(ctx, sessionKey(sessionID))
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return "", cache.ErrNotFound
		}
		return "", fmt.Errorf("session get: %w", err)
	}
	return string(data), nil
}
