package repo

import (
	"context"
	"fmt"

	"warehouse-controller/internal/outbox"
	"warehouse-controller/internal/platform/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgOutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) outbox.Repository {
	return &pgOutboxRepository{pool: pool}
}

func (r *pgOutboxRepository) Save(ctx context.Context, rec outbox.Record) error {
	q := postgres.Querier(ctx, r.pool)
	if _, err := q.Exec(ctx, `
		INSERT INTO outbox (topic, key, payload)
		VALUES ($1, $2, $3)
	`, rec.Topic, rec.Key, rec.Payload); err != nil {
		return fmt.Errorf("outbox save: %w", err)
	}
	return nil
}

func (r *pgOutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]outbox.Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, topic, key, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox fetch: %w", err)
	}
	defer rows.Close()

	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		if err := rows.Scan(&m.ID, &m.Topic, &m.Key, &m.Payload); err != nil {
			return nil, fmt.Errorf("outbox scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox rows: %w", err)
	}
	return msgs, nil
}

func (r *pgOutboxRepository) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
		UPDATE outbox SET published_at = NOW()
		WHERE id = ANY($1)
	`, ids); err != nil {
		return fmt.Errorf("outbox mark published: %w", err)
	}
	return nil
}

func (r *pgOutboxRepository) MarkFailed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
		UPDATE outbox SET attempts = attempts + 1
		WHERE id = ANY($1)
	`, ids); err != nil {
		return fmt.Errorf("outbox mark failed: %w", err)
	}
	return nil
}
