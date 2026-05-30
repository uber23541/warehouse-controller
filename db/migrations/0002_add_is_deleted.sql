-- +goose Up
ALTER TABLE products ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE products SET is_deleted = TRUE WHERE deleted_at IS NOT NULL;

-- +goose Down
ALTER TABLE products DROP COLUMN is_deleted;
