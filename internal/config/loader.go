package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

func Load[T any](path string, cfg *T) error {
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// I think it's not need
	// if err := cleanenv.ReadEnv(cfg); err != nil {
	// 	stdLog.Fatalf("Failed load env: %v", err)
	// }

	return nil
}
