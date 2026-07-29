package config

type TypeProxy string

const (
	TypeProxyHTTP   = "http"
	TypeProxySocks5 = "socks5"
)

type Proxy struct {
	Enable bool      `toml:"enable" env:"GIT_GEN_PROXY_ENABLE"`
	Typ    TypeProxy `toml:"type" env:"GIT_GEN_PROXY_TYPE"`
	Addr   string    `toml:"addr" env:"GIT_GEN_PROXY_ADDR"`
	Login  string    `toml:"login" env:"GIT_GEN_PROXY_LOGIN"`
	Passw  string    `toml:"passw" env:"GIT_GEN_PROXY_PASSW"`
}
