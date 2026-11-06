package generator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yvv4git/git-commit-gen/internal/config"
)

const testLimitsConfig = `
[generator]
baseBranch = "main"
rulesFile = "rules.md"
maxDiffBytes = 1000
maxRulesBytes = 2000
maxContextBytes = 3000
maxAgentTurns = 4
maxToolCallsPerTurn = 2

[[llms]]
name = "default"
type = "open-ai"
`

const testConfig = `
[generator]
baseBranch = "main"
rulesFile = "rules.md"

[[llms]]
name = "default"
type = "open-ai"

[llms.proxy]
enable = false
type = "socks5"
addr = "localhost:1080"
login = ""
passw = ""

[llms.openai]
api = "https://config.example/v1"
token = "toml-token"
model = "toml-model"

[llms.retry]
retries = 1
initInterval = "1s"
maxInterval = "2s"
`

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestLoadOverridesFromDotEnv(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.toml")
	writeTestFile(t, configPath, testConfig)
	writeTestFile(t, filepath.Join(configDir, ".env"), "GIT_GEN_BASE_BRANCH=wrong\n")

	dotEnv := `GIT_GEN_BASE_BRANCH=dev
GIT_GEN_PROXY_ENABLE=true
GIT_GEN_PROXY_TYPE=http
GIT_GEN_PROXY_ADDR=proxy.example:8080
GIT_GEN_PROXY_LOGIN=user
GIT_GEN_PROXY_PASSW=pass
GIT_GEN_LLM_RETRY_RETRIES=5
GIT_GEN_LLM_RETRY_INIT_INTERVAL=3s
GIT_GEN_LLM_RETRY_MAX_INTERVAL=4s
GIT_GEN_OPENAI_API=https://attacker.example/v1
GIT_GEN_OPENAI_TOKEN=env-token
GIT_GEN_OPENAI_MODEL=env-model
GIT_GEN_DOTENV_MARKER=secret
`
	if err := os.WriteFile(filepath.Join(workDir, ".env"), []byte(dotEnv), 0o600); err != nil {
		t.Fatal(err)
	}

	var cfg Config
	require.NoError(t, Load(configPath, &cfg))
	require.Equal(t, "dev", cfg.Gen.BaseBranch)
	require.Len(t, cfg.LLMs, 1)
	require.True(t, cfg.LLMs[0].Proxy.Enable)
	require.Equal(t, config.TypeProxy(config.TypeProxyHTTP), cfg.LLMs[0].Proxy.Typ)
	require.Equal(t, "proxy.example:8080", cfg.LLMs[0].Proxy.Addr)
	require.Equal(t, "user", cfg.LLMs[0].Proxy.Login)
	require.Equal(t, "pass", cfg.LLMs[0].Proxy.Passw)
	require.Equal(t, uint64(5), cfg.LLMs[0].Retry.Retries)
	require.Equal(t, 3*time.Second, cfg.LLMs[0].Retry.InitInterval)
	require.Equal(t, 4*time.Second, cfg.LLMs[0].Retry.MaxInterval)
	require.Equal(t, "https://config.example/v1", cfg.LLMs[0].OpenAI.API)
	require.Equal(t, "env-token", cfg.LLMs[0].OpenAI.Token)
	require.Equal(t, "env-model", cfg.LLMs[0].OpenAI.Model)
	require.Empty(t, os.Getenv("GIT_GEN_DOTENV_MARKER"))
}

func TestLoadIgnoresProcessEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_GEN_BASE_BRANCH", "hacked")
	t.Setenv("GIT_GEN_PROXY_ENABLE", "true")
	t.Setenv("GIT_GEN_LLM_RETRY_RETRIES", "99")
	t.Setenv("GIT_GEN_OPENAI_TOKEN", "hacked-token")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, configPath, testConfig)

	var cfg Config
	require.NoError(t, Load(configPath, &cfg))
	require.Equal(t, "main", cfg.Gen.BaseBranch)
	require.False(t, cfg.LLMs[0].Proxy.Enable)
	require.Equal(t, uint64(1), cfg.LLMs[0].Retry.Retries)
	require.Equal(t, "toml-token", cfg.LLMs[0].OpenAI.Token)
}

func TestLoadLeavesLimitsEmptyWhenAbsent(t *testing.T) {
	t.Chdir(t.TempDir())

	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, configPath, testConfig)

	var cfg Config
	require.NoError(t, Load(configPath, &cfg))
	require.Zero(t, cfg.Gen.MaxDiffBytes)
	require.Zero(t, cfg.Gen.MaxRulesBytes)
	require.Zero(t, cfg.Gen.MaxContextBytes)
	require.Zero(t, cfg.Gen.MaxAgentTurns)
	require.Zero(t, cfg.Gen.MaxToolCallsPerTurn)
}

func TestLoadReadsLimits(t *testing.T) {
	t.Chdir(t.TempDir())

	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, configPath, testLimitsConfig)

	var cfg Config
	require.NoError(t, Load(configPath, &cfg))
	require.Equal(t, 1000, cfg.Gen.MaxDiffBytes)
	require.Equal(t, 2000, cfg.Gen.MaxRulesBytes)
	require.Equal(t, 3000, cfg.Gen.MaxContextBytes)
	require.Equal(t, 4, cfg.Gen.MaxAgentTurns)
	require.Equal(t, 2, cfg.Gen.MaxToolCallsPerTurn)
}

func TestLoadRejectsInvalidDotEnvValue(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestFile(t, ".env", "GIT_GEN_LLM_RETRY_RETRIES=abc\n")

	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeTestFile(t, configPath, testConfig)

	var cfg Config
	err := Load(configPath, &cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "GIT_GEN_LLM_RETRY_RETRIES")
}
