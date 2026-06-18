package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTP  HTTPConfig
	Log   LogConfig
	DB    DBConfig
	Redis RedisConfig
	Auth  AuthConfig
	Kafka KafkaConfig
}

type HTTPConfig struct {
	Port string `env:"HTTP_PORT" env-default:"8080"`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
}

type DBConfig struct {
	DSN string `env:"DB_DSN" env-required:"true"`
}

type RedisConfig struct {
	URL string `env:"REDIS_URL" env-required:"true"`
}

type AuthConfig struct {
	JWTSecret  string        `env:"JWT_SECRET" env-required:"true"`
	AccessTTL  time.Duration `env:"ACCESS_TTL" env-default:"15m"`
	RefreshTTL time.Duration `env:"REFRESH_TTL" env-default:"168h"`
}

type KafkaConfig struct {
	Brokers       []string      `env:"KAFKA_BROKERS" env-default:"kafka:9092"`
	RelayInterval time.Duration `env:"OUTBOX_RELAY_INTERVAL" env-default:"2s"`
	RelayBatch    int           `env:"OUTBOX_RELAY_BATCH" env-default:"100"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
