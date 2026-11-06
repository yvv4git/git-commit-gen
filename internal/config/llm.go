package config

import "time"

type TypeClientLLM string

const (
	TypeClientLLMOpenAI = "open-ai"
)

type LLMs []LLM

type LLM struct {
	Name           string        `toml:"name"`
	Typ            TypeClientLLM `toml:"type"`
	Proxy          Proxy         `toml:"proxy"`
	OpenAI         OpenAI        `toml:"openai"`
	Retry          Retry         `toml:"retry"`
	RequestTimeout time.Duration `toml:"requestTimeout"`
}

type OpenAI struct {
	API               string `toml:"api"`
	Token             string `toml:"token"`
	Model             string `toml:"model"`
	AllowInsecureHTTP bool   `toml:"allowInsecureHttp"`
}

type Retry struct {
	Retries      uint64        `toml:"retries"`
	InitInterval time.Duration `toml:"initInterval"`
	MaxInterval  time.Duration `toml:"maxInterval"`
}
