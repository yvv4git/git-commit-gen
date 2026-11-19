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

// +LLM03:2025 Supply Chain
// +ASI04:2026 Agentic Supply Chain Vulnerabilities
// +T17 Supply Chain Compromise
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
