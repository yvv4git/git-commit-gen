package infra

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/tmc/langchaingo/llms/openai"
	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"golang.org/x/net/proxy"
)

func SetupOpenAIClient(cfg config.LLM) (*openai.LLM, error) {
	opts := []openai.Option{
		openai.WithBaseURL(cfg.OpenAI.API),
		openai.WithToken(cfg.OpenAI.Token),
		openai.WithModel(cfg.OpenAI.Model),
	}

	if cfg.Proxy.Enable {
		client, err := newHTTPClient(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("create proxy client: %w", err)
		}
		opts = append(opts, openai.WithHTTPClient(client))
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("setup openai client: %w", err)
	}

	return llm, nil
}

func newHTTPClient(p config.Proxy) (*http.Client, error) {
	var transport http.Transport

	switch p.Typ {
	case config.TypeProxyHTTP:
		proxyURL, err := url.Parse(p.Addr)
		if err != nil {
			return nil, fmt.Errorf("parse proxy addr: %w", err)
		}
		if p.Login != "" {
			proxyURL.User = url.UserPassword(p.Login, p.Passw)
		}
		transport.Proxy = http.ProxyURL(proxyURL)

	case config.TypeProxySocks5:
		var auth *proxy.Auth
		if p.Login != "" {
			auth = &proxy.Auth{User: p.Login, Password: p.Passw}
		}
		dialer, err := proxy.SOCKS5("tcp", p.Addr, auth, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("setup socks5 proxy: %w", err)
		}
		transport.DialContext = dialer.(proxy.ContextDialer).DialContext

	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedProxyType, p.Typ)
	}

	return &http.Client{Transport: &transport}, nil
}
