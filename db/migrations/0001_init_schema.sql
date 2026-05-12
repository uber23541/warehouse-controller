-- +goose Up
CREATE TABLE IF NOT EXISTS products (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    product_name    TEXT NOT NULL,
    manufacturer    TEXT NOT NULL,
    category        TEXT NOT NULL,

    count           INTEGER NOT NULL,
    price           BIGINT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS products;
