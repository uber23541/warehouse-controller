package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"warehouse-controller/internal/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const shutdownTimeout = 30 * time.Second

type App struct {
	logger      *zap.Logger
	server      *http.Server
	pool        *pgxpool.Pool
	relay       *outbox.Relay
	kafkaWriter *kafka.Writer
	relayWG     sync.WaitGroup
}

func (a *App) Run(ctx context.Context) {
	go func() {
		a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("app stopped with error", zap.Error(err))
		}
	}()

	a.relayWG.Add(1)
	go func() {
		defer a.relayWG.Done()
		a.relay.Run(ctx)
	}()

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

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	downs := []func(context.Context) error{
		a.server.Shutdown,
		func(context.Context) error { a.relayWG.Wait(); return nil },
		func(context.Context) error { return a.kafkaWriter.Close() },
		func(context.Context) error { a.pool.Close(); return nil },
	}
	for _, task := range downs {
		if err := task(ctx); err != nil {
			errs = append(errs, fmt.Errorf("GracefulShutdownError: %w", err))
		}
	}

	a.logger.Info("graceful shutdown completed")
	return errs
}
