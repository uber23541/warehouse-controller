-- +goose Up
ALTER TABLE outbox ADD COLUMN dead_at TIMESTAMPTZ;

DROP INDEX idx_outbox_unpublished;
CREATE INDEX idx_outbox_unpublished ON outbox (id) WHERE published_at IS NULL AND dead_at IS NULL;

-- +goose Down
DROP INDEX idx_outbox_unpublished;
CREATE INDEX idx_outbox_unpublished ON outbox (id) WHERE published_at IS NULL;

ALTER TABLE outbox DROP COLUMN dead_at;
