package generator

import "os"

const envBaseBranch = "GIT_GEN_BASE_BRANCH"

// -LLM01:2025 Prompt Injection: CWD .env can change base branch
// -LLM03:2025 Supply Chain: CWD .env is trusted
// -ASI01:2026 Agent Goal Hijack: CWD .env can change base branch
// -ASI04:2026 Agentic Supply Chain Vulnerabilities: CWD .env is trusted
// -T14 Human Attacks on Multi-Agent Systems: CWD .env is trusted
// -T17 Supply Chain Compromise: CWD .env is trusted
func (cfg *Config) applyDotEnv(values map[string]string) error {
	if value, ok := values[envBaseBranch]; ok && value != "" {
		cfg.Gen.BaseBranch = value
	}

	if value, ok := os.LookupEnv(envBaseBranch); ok && value != "" {
		cfg.Gen.BaseBranch = value
	}

	return nil
}
