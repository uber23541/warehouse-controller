package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
	server *http.Server
}

func (a *App) Run(ctx context.Context) {
	go func() {
		a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("app stopped with error", zap.Error(err))
		}
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

	downs := []func(context.Context) error{
		a.server.Shutdown,
	}
	for _, task := range downs {
		if err := task(context.Background()); err != nil {
			errs = append(errs, fmt.Errorf("GracefulShutdownError: %w", err))
		}
	}

	a.logger.Info("graceful shutdown completed")
	return errs
}
