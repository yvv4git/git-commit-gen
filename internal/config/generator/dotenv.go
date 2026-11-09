package generator

const envBaseBranch = "GIT_GEN_BASE_BRANCH"

func (cfg *Config) applyDotEnv(values map[string]string) error {
	if value, ok := values[envBaseBranch]; ok {
		cfg.Gen.BaseBranch = value
	}

	return nil
}
