// Package outbox реализует transactional outbox: события пишутся в таблицу outbox
// в одной транзакции с доменными изменениями (через postgres.Querier из контекста),
// а relay-воркер асинхронно перекладывает их в Kafka.
package outbox

import (
	"context"
	"fmt"

	"warehouse-controller/internal/platform/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Record — событие, готовое к записи в outbox.
type Record struct {
	Topic   string
	Key     string
	Payload []byte
}

// Message — строка outbox, ожидающая отправки в брокер.
type Message struct {
	ID      int64
	Topic   string
	Key     string
	Payload []byte
}

// Store пишет события в outbox и обслуживает их доставку.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Save вставляет событие в outbox через querier из контекста — то есть в текущую
// транзакцию, если она открыта TxManager.WithinTx (атомарно с доменной записью).
func (s *Store) Save(ctx context.Context, rec Record) error {
	q := postgres.Querier(ctx, s.pool)
	if _, err := q.Exec(ctx, `
		INSERT INTO outbox (topic, key, payload)
		VALUES ($1, $2, $3)
	`, rec.Topic, rec.Key, rec.Payload); err != nil {
		return fmt.Errorf("outbox save: %w", err)
	}
	return nil
}

// FetchUnpublished возвращает неотправленные сообщения в порядке возрастания id.
func (s *Store) FetchUnpublished(ctx context.Context, limit int) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `
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

	var msgs []Message
	for rows.Next() {
		var m Message
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

// MarkPublished помечает сообщения как доставленные.
func (s *Store) MarkPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE outbox SET published_at = NOW()
		WHERE id = ANY($1)
	`, ids); err != nil {
		return fmt.Errorf("outbox mark published: %w", err)
	}
	return nil
}

// MarkFailed увеличивает счётчик попыток доставки.
func (s *Store) MarkFailed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE outbox SET attempts = attempts + 1
		WHERE id = ANY($1)
	`, ids); err != nil {
		return fmt.Errorf("outbox mark failed: %w", err)
	}
	return nil
}
