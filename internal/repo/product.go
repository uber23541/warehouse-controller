package repo

import (
	"context"
	"fmt"

	"warehouse-controller/internal/metrics"
	"warehouse-controller/internal/repo/dbmodel"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository interface {
	Create(ctx context.Context, product *dbmodel.Product) (int64, error)
	GetByID(ctx context.Context, id int64) (*dbmodel.Product, error)
	Delete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) (*dbmodel.Product, error)
	Search(ctx context.Context, filter dbmodel.ProductFilter) ([]dbmodel.Product, error)
	Patch(ctx context.Context, id int64, patch dbmodel.ProductPatch) (*dbmodel.Product, error)
	List(ctx context.Context, limit, offset int32) ([]dbmodel.Product, error)
}

type pgProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepo(pool *pgxpool.Pool) ProductRepository {
	return &pgProductRepository{pool: pool}
}

func (r *pgProductRepository) Create(ctx context.Context, product *dbmodel.Product) (int64, error) {
	defer metrics.ObserveDBQuery("create")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO products (product_name, manufacturer, category, count, price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, product.ProductName, product.Manufacturer, product.Category, product.Count, product.Price).Scan(&id)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return id, nil
}

func (r *pgProductRepository) GetByID(ctx context.Context, id int64) (*dbmodel.Product, error) {
	defer metrics.ObserveDBQuery("get_by_id")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	var p dbmodel.Product
	err = tx.QueryRow(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &p, nil
}

func (r *pgProductRepository) Delete(ctx context.Context, id int64) error {
	defer metrics.ObserveDBQuery("delete")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(ctx, `
		UPDATE products
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *pgProductRepository) Restore(ctx context.Context, id int64) (*dbmodel.Product, error) {
	defer metrics.ObserveDBQuery("restore")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	var p dbmodel.Product
	err = tx.QueryRow(ctx, `
		UPDATE products
		SET deleted_at = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NOT NULL
		RETURNING id, product_name, manufacturer, category, count, price
	`, id).Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &p, nil
}

func (r *pgProductRepository) Search(ctx context.Context, filter dbmodel.ProductFilter) ([]dbmodel.Product, error) {
	defer metrics.ObserveDBQuery("search")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	rows, err := tx.Query(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE deleted_at IS NULL
		  AND ($1::text IS NULL OR product_name ILIKE '%' || $1::text || '%')
		  AND ($2::text IS NULL OR manufacturer = $2::text)
		  AND ($3::text IS NULL OR category = $3::text)
		  AND ($4::bigint IS NULL OR price >= $4::bigint)
		  AND ($5::bigint IS NULL OR price <= $5::bigint)
		ORDER BY id DESC
		LIMIT $6 OFFSET $7
	`, filter.ProductName, filter.Manufacturer, filter.Category, filter.MinPrice, filter.MaxPrice, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dbmodel.Product
	for rows.Next() {
		var p dbmodel.Product
		if err := rows.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return products, nil
}

func (r *pgProductRepository) Patch(ctx context.Context, id int64, patch dbmodel.ProductPatch) (*dbmodel.Product, error) {
	defer metrics.ObserveDBQuery("patch")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	var p dbmodel.Product
	err = tx.QueryRow(ctx, `
		UPDATE products
		SET
			product_name = COALESCE($1::text, product_name),
			manufacturer = COALESCE($2::text, manufacturer),
			category = COALESCE($3::text, category),
			count = COALESCE($4::integer, count),
			price = COALESCE($5::bigint, price),
			updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING id, product_name, manufacturer, category, count, price
	`, patch.ProductName, patch.Manufacturer, patch.Category, patch.Count, patch.Price, id).Scan(
		&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &p, nil
}

func (r *pgProductRepository) List(ctx context.Context, limit, offset int32) ([]dbmodel.Product, error) {
	defer metrics.ObserveDBQuery("list")()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	rows, err := tx.Query(ctx, `
		SELECT id, product_name, manufacturer, category, count, price
		FROM products
		WHERE deleted_at IS NULL
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []dbmodel.Product
	for rows.Next() {
		var p dbmodel.Product
		if err := rows.Scan(&p.ID, &p.ProductName, &p.Manufacturer, &p.Category, &p.Count, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return products, nil
}
