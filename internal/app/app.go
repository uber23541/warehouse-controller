package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"warehouse-controller/internal/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type App struct {
	logger      *zap.Logger
	server      *http.Server
	pool        *pgxpool.Pool
	relay       *outbox.Relay
	kafkaWriter *kafka.Writer
}

func (a *App) Run(ctx context.Context) {
	go func() {
		a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("app stopped with error", zap.Error(err))
		}
	}()

	go a.relay.Run(ctx)

	<-ctx.Done()

	for _, err := range a.shutdown() {
		if !errors.Is(err, context.Canceled) {
			a.logger.Error("shutdown error", zap.Error(err))
		}
	}
	a.logger.Info("app stopped")
}

func (a *App) shutdown() []error {
	a.logger.Info("starting graceful shutdown")
	var errs []error

	downs := []func(context.Context) error{
		a.server.Shutdown,
		func(context.Context) error { return a.kafkaWriter.Close() },
		func(context.Context) error { a.pool.Close(); return nil },
	}
	for _, task := range downs {
		if err := task(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("GracefulShutdownError: %w", err))
		}
	}

	a.logger.Info("graceful shutdown completed")
	return errs
}
