package config

type TypeProxy string

const (
	TypeProxyHTTP   = "http"
	TypeProxySocks5 = "socks5"
)

type Proxy struct {
	Enable bool      `toml:"enable"`
	Typ    TypeProxy `toml:"type"`
	Addr   string    `toml:"addr"`
	Login  string    `toml:"login"`
	Passw  string    `toml:"passw"`
}
