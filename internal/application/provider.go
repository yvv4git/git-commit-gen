package application

import (
	"errors"
	"fmt"

	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/config/generator"
)

func selectLLMProvider(cfg generator.Config, name string) (config.LLM, error) {
	if len(cfg.LLMs) == 0 {
		return config.LLM{}, errors.New("no LLM providers configured")
	}

	if name == "" {
		return cfg.LLMs[0], nil
	}

	for _, llmCfg := range cfg.LLMs {
		if llmCfg.Name == name {
			return llmCfg, nil
		}
	}

	return config.LLM{}, fmt.Errorf("LLM provider %q not found", name)
}
