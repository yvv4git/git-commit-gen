package infra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/tmc/langchaingo/llms/openai"
	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

type modelLoggingTransport struct {
	base http.RoundTripper
	log  *zap.Logger
}

type modelResponse struct {
	Model string `json:"model"`
}

func SetupOpenAIClient(cfg config.LLM, log *zap.Logger) (*openai.LLM, error) {
	client, err := newHTTPClient(cfg.Proxy, log)
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	opts := []openai.Option{
		openai.WithBaseURL(cfg.OpenAI.API),
		openai.WithToken(cfg.OpenAI.Token),
		openai.WithModel(cfg.OpenAI.Model),
		openai.WithHTTPClient(client),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("setup openai client: %w", err)
	}

	return llm, nil
}

func newHTTPClient(p config.Proxy, log *zap.Logger) (*http.Client, error) {
	if !p.Enable {
		return &http.Client{Transport: &modelLoggingTransport{base: http.DefaultTransport, log: log}}, nil
	}

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

	return &http.Client{Transport: &modelLoggingTransport{base: &transport, log: log}}, nil
}

func (t *modelLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return resp, nil
	}

	originalBody := resp.Body
	body, readErr := io.ReadAll(originalBody)
	closeErr := originalBody.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	if readErr != nil {
		return nil, fmt.Errorf("read response body: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close response body: %w", closeErr)
	}

	t.logModel(body)

	return resp, nil
}

func (t *modelLoggingTransport) logModel(body []byte) {
	var payload modelResponse
	if json.Unmarshal(body, &payload) != nil || payload.Model == "" {
		return
	}

	t.log.Info("LLM model", zap.String("model", payload.Model))
}
