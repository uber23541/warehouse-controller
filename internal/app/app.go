package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"warehouse-controller/internal/config"
	"warehouse-controller/internal/handler"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type App struct {
	logger  *zap.Logger
	handler *handler.WarehouseHandler
	cfg     config.Config
	server  *http.Server
}

func New(logger *zap.Logger, h *handler.WarehouseHandler, cfg config.Config) *App {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))
	h.RegisterRoutes(r)

	return &App{
		logger:  logger,
		handler: h,
		cfg:     cfg,
	}
}

func (a *App) ListenAndServe() error {
	a.logger.Info("starting HTTP server", zap.String("address", a.server.Addr))
	return a.server.ListenAndServe()
}

func (a *App) Shutdown() []error {
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

func (a *App) Logger() *zap.Logger {
	return a.logger
}
