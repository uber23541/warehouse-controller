package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DB_DSN    string `env:"DB_DSN" env-required:"true"`
	HTTP_PORT string `env:"HTTP_PORT" env-default:"8080"`
	REDIS_URL string `env:"REDIS_URL" env-required:"true"`
	LOG_LEVEL string `env:"LOG_LEVEL" env-default:"info"`
}

func Load() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
