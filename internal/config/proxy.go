package config

type TypeProxy string

const (
	TypeProxyHTTP   = "http"
	TypeProxySocks5 = "socks5"
)

type Proxy struct {
	Enable bool      `toml:"enable"`
	Typ    TypeProxy `toml:"type" env:"PROXY_TYPE"`
	Addr   string    `toml:"addr" env:"PROXY_ADDR"`
	Login  string    `toml:"login" env:"PROXY_LOGIN"`
	Passw  string    `toml:"passw" env:"PROXY_PASSW"`
}
