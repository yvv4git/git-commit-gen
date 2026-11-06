package generator

import (
	"github.com/yvv4git/git-commit-gen/internal/config"
)

type Config struct {
	Log  config.Log  `toml:"log"`
	Gen  Generator   `toml:"generator"`
	LLMs config.LLMs `toml:"llms"`
}

type Generator struct {
	BaseBranch          string `toml:"baseBranch"`
	RulesFile           string `toml:"rulesFile"`
	MaxDiffBytes        int    `toml:"maxDiffBytes"`
	MaxRulesBytes       int    `toml:"maxRulesBytes"`
	MaxContextBytes     int    `toml:"maxContextBytes"`
	MaxAgentTurns       int    `toml:"maxAgentTurns"`
	MaxToolCallsPerTurn int    `toml:"maxToolCallsPerTurn"`
}

func Load(path string, cfg *Config) error {
	if err := config.Load(path, cfg); err != nil {
		return err
	}

	values, err := config.LoadDotEnv()
	if err != nil {
		return err
	}

	return cfg.applyDotEnv(values)
}
