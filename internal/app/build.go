package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/cache"
	"warehouse-controller/internal/config"
	"warehouse-controller/internal/handler"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func RunMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db, "db/migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func Build(ctx context.Context, cfg config.Config, logger *zap.Logger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.DB.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	logger.Info("database connected")
	productRepo := repo.NewProductRepo(pool)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.URL})
	logger.Info("redis client created")
	productCache := cache.New(rdb)
	sessionRepo := repo.NewSessionRepo(rdb)

	issuer := auth.NewIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)

	warehouseSvc := service.NewWarehouseService(productRepo, productCache, logger)
	authSvc := service.NewAuthService(issuer, sessionRepo)

	warehouseH := handler.NewWarehouseHandler(warehouseSvc, logger)
	authH := handler.NewAuthHandler(authSvc, logger)

	r := handler.BuildRouter(warehouseH, authH, issuer, logger)

	return &App{
		logger: logger,
		server: &http.Server{
			Addr:    ":" + cfg.HTTP.Port,
			Handler: r,
		},
	}, nil
}
