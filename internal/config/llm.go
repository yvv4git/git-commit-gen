package config

type TypeClientLLM string

const (
	TypeClientLLMOpenAI = "open-ai"
)

type LLM struct {
	Typ    TypeClientLLM `toml:"type"`
	Proxy  Proxy         `toml:"proxy"`
	OpenAI OpenAI        `toml:"openai"`
}

type OpenAI struct {
	API   string `toml:"api" env:"OPENAI_API"`
	Token string `toml:"token" env:"OPENAI_TOKEN"`
	Model string `toml:"model" env:"OPENAI_MODEL"`
}
