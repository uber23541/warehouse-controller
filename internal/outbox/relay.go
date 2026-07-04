package outbox

import (
	"context"
	"fmt"
	"time"

	"warehouse-controller/internal/platform/postgres"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Relay периодически вычитывает неотправленные строки outbox и публикует их в Kafka.
type Relay struct {
	store    Repository
	txm      postgres.Transactor
	writer   *kafka.Writer
	log      *zap.Logger
	interval time.Duration
	batch    int
}

func NewRelay(store Repository, txm postgres.Transactor, writer *kafka.Writer, log *zap.Logger, interval time.Duration, batch int) *Relay {
	return &Relay{store: store, txm: txm, writer: writer, log: log, interval: interval, batch: batch}
}

// Run крутит цикл доставки до отмены контекста.
func (r *Relay) Run(ctx context.Context) {
	r.log.Info("outbox relay started", zap.Duration("interval", r.interval), zap.Int("batch", r.batch))
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.log.Info("outbox relay stopped")
			return
		case <-ticker.C:
			r.processBatch(ctx)
		}
	}
}

// processBatch выполняет fetch+publish+mark в одной транзакции
func (r *Relay) processBatch(ctx context.Context) {
	err := r.txm.WithinTx(ctx, func(ctx context.Context) error {
		msgs, err := r.store.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}
		if len(msgs) == 0 {
			return nil
		}

		kmsgs := make([]kafka.Message, len(msgs))
		ids := make([]int64, len(msgs))
		for i, m := range msgs {
			kmsgs[i] = kafka.Message{Topic: m.Topic, Key: []byte(m.Key), Value: m.Payload}
			ids[i] = m.ID
		}

		if err := r.writer.WriteMessages(ctx, kmsgs...); err != nil {
			r.log.Warn("outbox publish failed", zap.Int("count", len(msgs)), zap.Error(err))
			dead, mErr := r.store.MarkFailed(ctx, ids)
			if mErr != nil {
				return fmt.Errorf("mark failed: %w", mErr)
			}
			if dead > 0 {
				r.log.Error("outbox messages dead-lettered", zap.Int64("count", dead))
			}
			return nil
		}

		if err := r.store.MarkPublished(ctx, ids); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
		r.log.Debug("outbox batch published", zap.Int("count", len(msgs)))
		return nil
	})
	if err != nil {
		r.log.Error("outbox batch failed", zap.Error(err))
	}
}
