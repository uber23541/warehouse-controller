-- name: CreateProduct :one
INSERT INTO products (
    product_name,
    manufacturer,
    category,
    count,
    price
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetProductByID :one
SELECT *
FROM products
WHERE id = $1
  AND deleted_at IS NULL;

-- name: ListProducts :many
SELECT *
FROM products
WHERE deleted_at IS NULL
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: PatchProduct :one
UPDATE products
SET
    product_name = COALESCE(sqlc.narg('product_name')::text, product_name),
    manufacturer = COALESCE(sqlc.narg('manufacturer')::text, manufacturer),
    category = COALESCE(sqlc.narg('category')::text, category),
    count = COALESCE(sqlc.narg('count')::integer, count),
    price = COALESCE(sqlc.narg('price')::bigint, price),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteProduct :exec
UPDATE products
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: RestoreProduct :one
UPDATE products
SET
    deleted_at = NULL,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NOT NULL
RETURNING *;

-- name: SearchProducts :many
SELECT *
FROM products
WHERE deleted_at IS NULL
  AND (
    sqlc.narg('product_name')::text IS NULL
    OR product_name ILIKE '%' || sqlc.narg('product_name')::text || '%'
  )
  AND (
    sqlc.narg('manufacturer')::text IS NULL
    OR manufacturer = sqlc.narg('manufacturer')::text
  )
  AND (
    sqlc.narg('category')::text IS NULL
    OR category = sqlc.narg('category')::text
  )
  AND (
    sqlc.narg('min_price')::bigint IS NULL
    OR price >= sqlc.narg('min_price')::bigint
  )
  AND (
    sqlc.narg('max_price')::bigint IS NULL
    OR price <= sqlc.narg('max_price')::bigint
  )
ORDER BY id DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;