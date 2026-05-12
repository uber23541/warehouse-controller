package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"warehouse-controller/internal/app"
	"warehouse-controller/internal/config"
	"warehouse-controller/internal/handler"
	"warehouse-controller/internal/service"
	"warehouse-controller/internal/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config.Load: %v", err)
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.LOG_LEVEL)); err != nil {
		level = zapcore.InfoLevel
	}

	zapLog, err := zap.NewProduction(zap.IncreaseLevel(level))
	if err != nil {
		log.Fatalf("zap.NewProduction: %v", err)
	}
	defer zapLog.Sync()

	// migrations
	db, err := sql.Open("pgx", cfg.DB_DSN)
	if err != nil {
		zapLog.Fatal("failed to open db for migrations", zap.Error(err))
	}
	if err := goose.SetDialect("postgres"); err != nil {
		zapLog.Fatal("goose dialect", zap.Error(err))
	}
	if err := goose.Up(db, "db/migrations"); err != nil {
		zapLog.Fatal("goose up", zap.Error(err))
	}
	db.Close()
	zapLog.Info("migrations applied")

	// database pool
	pool, err := pgxpool.New(ctx, cfg.DB_DSN)
	if err != nil {
		zapLog.Fatal("failed to connect database", zap.Error(err))
	}
	defer pool.Close()
	zapLog.Info("database connected")

	// dependencies
	queries := sqlc.New(pool)
	svc := service.NewWarehouseService(pool, queries, zapLog)
	h := handler.NewWarehouseHandler(svc, zapLog)

	// app
	a := app.New(zapLog, h, cfg)

	go func() {
		if err := a.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger().Error("app stopped with error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	errs := a.Shutdown()
	for _, err := range errs {
		if err != context.Canceled {
			a.Logger().Error("shutdown error", zap.Error(err))
		}
	}

	a.Logger().Info("app stopped")
}
