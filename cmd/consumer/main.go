// Сервис-консьюмер: вычитывает события из Kafka и логирует их.
// Это заглушка-сток — её задача не дать сообщениям копиться в брокере,
// поскольку проект изолирован и реального потребителя событий пока нет.
package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type config struct {
	Brokers []string `env:"KAFKA_BROKERS" env-default:"kafka:9092"`
	Topics  []string `env:"KAFKA_TOPICS" env-default:"product.created,product.deleted"`
	GroupID string   `env:"KAFKA_GROUP_ID" env-default:"warehouse-sink"`
	Level   string   `env:"LOG_LEVEL" env-default:"info"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var cfg config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("config: %v", err)
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	logger, err := zap.NewProduction(zap.IncreaseLevel(level))
	if err != nil {
		log.Fatalf("zap.NewProduction: %v", err)
	}
	defer logger.Sync()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.GroupID,
		GroupTopics: cfg.Topics,
	})
	defer reader.Close()

	logger.Info("consumer started",
		zap.Strings("brokers", cfg.Brokers),
		zap.Strings("topics", cfg.Topics),
		zap.String("group", cfg.GroupID),
	)

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("consumer stopped")
				return
			}
			logger.Error("read message failed", zap.Error(err))
			continue
		}

		logger.Info("event received",
			zap.String("topic", msg.Topic),
			zap.String("key", string(msg.Key)),
			zap.String("payload", strings.TrimSpace(string(msg.Value))),
			zap.Int("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)
	}
}
