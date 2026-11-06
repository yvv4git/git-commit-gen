package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const DotEnvFileName = ".env"

func Load[T any](path string, cfg *T) error {
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

func LoadDotEnv() (map[string]string, error) {
	values, err := godotenv.Read(DotEnvFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read .env: %w", err)
	}

	return values, nil
}
