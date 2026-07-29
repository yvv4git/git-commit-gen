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
	API   string `toml:"api" env:"GIT_GEN_OPENAI_API"`
	Token string `toml:"token" env:"GIT_GEN_OPENAI_TOKEN"`
	Model string `toml:"model" env:"GIT_GEN_OPENAI_MODEL"`
}
