package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type SetupParams struct {
	ConfigContent string
	RulesContent  string
}

type Setup struct {
	configDir ports.ConfigPath
	writer    ports.FileWriter
}

func NewSetup(configDir ports.ConfigPath, writer ports.FileWriter) *Setup {
	return &Setup{
		configDir: configDir,
		writer:    writer,
	}
}

func (s *Setup) Run(ctx context.Context, params *SetupParams) error {
	dir, err := s.configDir.DefaultConfigDir(ctx)
	if err != nil {
		return fmt.Errorf("get config dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := s.writer.WriteFile(ctx, &ports.WriteFileParams{
		FilePath: filepath.Join(dir, "config.toml"),
		Content:  params.ConfigContent,
	}); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := s.writer.WriteFile(ctx, &ports.WriteFileParams{
		FilePath: filepath.Join(dir, "rules.md"),
		Content:  params.RulesContent,
	}); err != nil {
		return fmt.Errorf("write rules: %w", err)
	}

	return nil
}
