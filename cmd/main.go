package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"warehouse-controller/internal/app"
	"warehouse-controller/internal/config"

	_ "warehouse-controller/docs"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// @title           Warehouse Controller API
// @version         1.0
// @description     API для управления складом
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config.Load: %v", err)
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	logger, err := zap.NewProduction(zap.IncreaseLevel(level))
	if err != nil {
		log.Fatalf("zap.NewProduction: %v", err)
	}
	defer logger.Sync()

	if err := app.RunMigrations(cfg.DB.DSN); err != nil {
		logger.Fatal("migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	a, err := app.Build(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("app.Build", zap.Error(err))
	}

	a.Run(ctx)
}
