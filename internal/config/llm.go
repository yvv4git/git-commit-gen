package config

import "time"

type TypeClientLLM string

const (
	TypeClientLLMOpenAI = "open-ai"
)

type LLMs []LLM

type LLM struct {
	Name   string        `toml:"name"`
	Typ    TypeClientLLM `toml:"type"`
	Proxy  Proxy         `toml:"proxy"`
	OpenAI OpenAI        `toml:"openai"`
	Retry  Retry         `toml:"retry"`
}

type OpenAI struct {
	API   string `toml:"api"`
	Token string `toml:"token"`
	Model string `toml:"model"`
}

type Retry struct {
	Retries      uint64        `toml:"retries" env:"GIT_GEN_LLM_RETRY_RETRIES"`
	InitInterval time.Duration `toml:"initInterval" env:"GIT_GEN_LLM_RETRY_INIT_INTERVAL"`
	MaxInterval  time.Duration `toml:"maxInterval" env:"GIT_GEN_LLM_RETRY_MAX_INTERVAL"`
}
