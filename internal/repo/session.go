package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionRepo struct {
	client *redis.Client
}

func NewSessionRepo(client *redis.Client) *SessionRepo {
	return &SessionRepo{client: client}
}

func sessionKey(sessionID string) string {
	return "refresh:" + sessionID
}

func (r *SessionRepo) SetRefreshJTI(ctx context.Context, sessionID, jti string, ttl time.Duration) error {
	if err := r.client.Set(ctx, sessionKey(sessionID), jti, ttl).Err(); err != nil {
		return fmt.Errorf("session set: %w", err)
	}
	return nil
}

func (r *SessionRepo) GetRefreshJTI(ctx context.Context, sessionID string) (string, error) {
	jti, err := r.client.Get(ctx, sessionKey(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("session get: %w", err)
	}
	return jti, nil
}
