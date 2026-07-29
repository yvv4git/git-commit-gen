package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

func Load[T any](path string, cfg *T) error {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return fmt.Errorf("load .env: %w", err)
		}
	}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}
