package generator

import (
	"github.com/yvv4git/git-commit-gen/internal/config"
)

type Config struct {
	Log config.Log `toml:"log"`
	Gen Generator  `toml:"generator"`
	LLM config.LLM `toml:"llm"`
}

type Generator struct {
	BaseBranch string `toml:"baseBranch"`
	RulesFile  string `toml:"rulesFile"`
}

func Load(path string, cfg *Config) error {
	return config.Load(path, cfg)
}
