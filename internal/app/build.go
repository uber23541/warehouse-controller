package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"warehouse-controller/internal/auth"
	"warehouse-controller/internal/config"
	"warehouse-controller/internal/handler"
	"warehouse-controller/internal/outbox"
	"warehouse-controller/internal/platform/cache"
	sessioncache "warehouse-controller/internal/platform/cache/session"
	"warehouse-controller/internal/platform/postgres"
	"warehouse-controller/internal/repo"
	"warehouse-controller/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
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

	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}
	rdb := redis.NewClient(redisOpts)
	logger.Info("redis client created")
	sharedCache := cache.New(rdb)
	sessionStore := sessioncache.New(sharedCache)

	issuer := auth.NewIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, cfg.Auth.RefreshTTL)

	txManager := postgres.NewTxManager(pool)
	outboxRepo := repo.NewOutboxRepo(pool, cfg.Kafka.RelayMaxAttempts)

	kafkaWriter := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}
	relay := outbox.NewRelay(outboxRepo, txManager, kafkaWriter, logger, cfg.Kafka.RelayInterval, cfg.Kafka.RelayBatch)

	warehouseSvc := service.NewWarehouseService(productRepo, sharedCache, txManager, outboxRepo, logger)
	authSvc := service.NewAuthService(issuer, sessionStore)

	warehouseH := handler.NewWarehouseHandler(warehouseSvc, logger)
	authH := handler.NewAuthHandler(authSvc, logger)

	router := handler.NewRouter(warehouseH, authH, authSvc, logger)

	return &App{
		logger:      logger,
		pool:        pool,
		relay:       relay,
		kafkaWriter: kafkaWriter,
		server: &http.Server{
			Addr:    ":" + cfg.HTTP.Port,
			Handler: router.Engine(),
		},
	}, nil
}
