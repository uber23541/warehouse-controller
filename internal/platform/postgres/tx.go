// Package postgres содержит границу транзакции (Unit of Work) поверх pgx.
//
// Транзакция передаётся не через сигнатуры методов, а через context: TxManager.WithinTx
// открывает pgx.Tx и кладёт её в ctx, а репозитории и хранилища достают её через Querier.
// Это позволяет нескольким сущностям (репозиторий продукта и outbox-хранилище) писать
// в одной транзакции, не протекая типом pgx.Tx в их интерфейсы.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX — общий набор методов, который реализуют и *pgxpool.Pool, и pgx.Tx.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Transactor задаёт границу транзакции. Внедряется в сервис, мокируется в тестах.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txKey struct{}

// TxManager управляет транзакциями на пуле pgx.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithinTx открывает транзакцию, кладёт её в контекст и выполняет fn.
// Транзакция коммитится при успехе и откатывается при ошибке или панике.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.Background())
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()

	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Querier возвращает транзакцию из контекста, если она есть, иначе сам пул.
func Querier(ctx context.Context, pool DBTX) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
