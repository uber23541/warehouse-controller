package outbox

import "context"

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

type Repository interface {
	Save(ctx context.Context, rec Record) error
	FetchUnpublished(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, ids []int64) error
	MarkFailed(ctx context.Context, ids []int64) (deadCount int64, err error)
}
