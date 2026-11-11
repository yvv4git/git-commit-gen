package generator

import "os"

const envBaseBranch = "GIT_GEN_BASE_BRANCH"

func (cfg *Config) applyDotEnv(values map[string]string) error {
	if value, ok := values[envBaseBranch]; ok && value != "" {
		cfg.Gen.BaseBranch = value
	}

	if value, ok := os.LookupEnv(envBaseBranch); ok && value != "" {
		cfg.Gen.BaseBranch = value
	}

	return nil
}
