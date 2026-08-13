package config

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
}

type OpenAI struct {
	API   string `toml:"api"`
	Token string `toml:"token"`
	Model string `toml:"model"`
}
