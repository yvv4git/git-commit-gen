package config

type Log struct {
	Level string `toml:"level" env:"LOG_LEVEL"`
}
