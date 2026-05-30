package event

import "context"

type NoopPublisher struct{}

func NewNoopPublisher() Publisher {
	return NoopPublisher{}
}

func (NoopPublisher) Publish(_ context.Context, _ Event) error {
	return nil
}
