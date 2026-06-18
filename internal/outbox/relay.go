package outbox

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Relay периодически вычитывает неотправленные строки outbox и публикует их в Kafka.
type Relay struct {
	store    *Store
	writer   *kafka.Writer
	log      *zap.Logger
	interval time.Duration
	batch    int
}

func NewRelay(store *Store, writer *kafka.Writer, log *zap.Logger, interval time.Duration, batch int) *Relay {
	return &Relay{store: store, writer: writer, log: log, interval: interval, batch: batch}
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

func (r *Relay) processBatch(ctx context.Context) {
	msgs, err := r.store.FetchUnpublished(ctx, r.batch)
	if err != nil {
		r.log.Error("outbox fetch failed", zap.Error(err))
		return
	}
	if len(msgs) == 0 {
		return
	}

	kmsgs := make([]kafka.Message, len(msgs))
	ids := make([]int64, len(msgs))
	for i, m := range msgs {
		kmsgs[i] = kafka.Message{Topic: m.Topic, Key: []byte(m.Key), Value: m.Payload}
		ids[i] = m.ID
	}

	if err := r.writer.WriteMessages(ctx, kmsgs...); err != nil {
		r.log.Warn("outbox publish failed", zap.Int("count", len(msgs)), zap.Error(err))
		if err := r.store.MarkFailed(ctx, ids); err != nil {
			r.log.Error("outbox mark failed", zap.Error(err))
		}
		return
	}

	if err := r.store.MarkPublished(ctx, ids); err != nil {
		// Сообщения уже в Kafka, но пометить не удалось — будут переотправлены (at-least-once).
		r.log.Error("outbox mark published failed", zap.Int64s("ids", ids), zap.Error(err))
		return
	}
	r.log.Debug("outbox batch published", zap.Int("count", len(msgs)))
}
