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

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
