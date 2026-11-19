package setup

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type SetupParams struct {
	ConfigContent string
	RulesContent  string
	Force         bool
}

type SetupResult struct {
	Skipped []string
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

func (s *Setup) Run(ctx context.Context, params *SetupParams) (*SetupResult, error) {
	dir, err := s.configDir.DefaultConfigDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("get config dir: %w", err)
	}

	if err := s.writer.MkdirAll(ctx, &ports.MkdirAllParams{DirPath: dir}); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}

	result := &SetupResult{}
	configPath := filepath.Join(dir, "config.toml")
	writeConfig, err := s.shouldWrite(ctx, configPath, params.Force)
	if err != nil {
		return nil, fmt.Errorf("check config: %w", err)
	}
	if writeConfig {
		if err := s.writer.WriteFile(ctx, &ports.WriteFileParams{
			FilePath: configPath,
			Content:  params.ConfigContent,
		}); err != nil {
			return nil, fmt.Errorf("write config: %w", err)
		}
	} else {
		result.Skipped = append(result.Skipped, configPath)
	}

	rulesPath := filepath.Join(dir, "rules.md")
	writeRules, err := s.shouldWrite(ctx, rulesPath, params.Force)
	if err != nil {
		return nil, fmt.Errorf("check rules: %w", err)
	}
	if writeRules {
		if err := s.writer.WriteFile(ctx, &ports.WriteFileParams{
			FilePath: rulesPath,
			Content:  params.RulesContent,
		}); err != nil {
			return nil, fmt.Errorf("write rules: %w", err)
		}
	} else {
		result.Skipped = append(result.Skipped, rulesPath)
	}

	return result, nil
}

func (s *Setup) shouldWrite(ctx context.Context, path string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	exists, err := s.writer.FileExists(ctx, &ports.FileExistsParams{FilePath: path})
	if err != nil {
		return false, err
	}

	return !exists, nil
}
