package infra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"

	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/pkg/retry"
)

type RetryStatus struct {
	mu         sync.Mutex
	statusCode int
	retryAfter time.Duration
}

func (s *RetryStatus) Set(statusCode int, retryAfter time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.statusCode = statusCode
	s.retryAfter = retryAfter
}

func (s *RetryStatus) Take() (int, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	statusCode := s.statusCode
	retryAfter := s.retryAfter
	s.statusCode = 0
	s.retryAfter = 0

	return statusCode, retryAfter
}

type modelLoggingTransport struct {
	base        http.RoundTripper
	log         *zap.Logger
	retryStatus *RetryStatus
}

type modelResponse struct {
	Model string `json:"model"`
}

const defaultRequestTimeout = 90 * time.Second

func SetupLLMClient(cfg config.LLM, log *zap.Logger) (*openai.LLM, *RetryStatus, error) {
	if err := validateAPIURL(cfg.OpenAI.API, cfg.OpenAI.AllowInsecureHTTP); err != nil {
		return nil, nil, err
	}

	client, retryStatus, err := newHTTPClient(cfg.Proxy, cfg.RequestTimeout, log)
	if err != nil {
		return nil, nil, fmt.Errorf("create http client: %w", err)
	}

	opts := []openai.Option{
		openai.WithBaseURL(cfg.OpenAI.API),
		openai.WithToken(cfg.OpenAI.Token),
		openai.WithModel(cfg.OpenAI.Model),
		openai.WithHTTPClient(client),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("setup openai client: %w", err)
	}

	return llm, retryStatus, nil
}

func validateAPIURL(rawURL string, allowInsecureHTTP bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return domain.ErrInvalidAPIURL
	}

	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return nil
	case "http":
		if allowInsecureHTTP || isLoopbackHost(parsed.Hostname()) {
			return nil
		}

		return fmt.Errorf("%w: %s", domain.ErrInsecureAPIURL, parsed.Redacted())
	default:
		return fmt.Errorf("%w: unsupported scheme %q", domain.ErrInvalidAPIURL, parsed.Scheme)
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func newHTTPClient(p config.Proxy, requestTimeout time.Duration, log *zap.Logger) (*http.Client, *RetryStatus, error) {
	factory := transportFactory{proxy: p}

	transport, err := factory.build()
	if err != nil {
		return nil, nil, err
	}

	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}

	retryStatus := &RetryStatus{}
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &modelLoggingTransport{
			base:        transport,
			log:         log,
			retryStatus: retryStatus,
		},
	}

	return client, retryStatus, nil
}

type transportFactory struct {
	proxy config.Proxy
}

func (factory *transportFactory) build() (*http.Transport, error) {
	if !factory.proxy.Enable {
		return factory.direct()
	}

	switch factory.proxy.Typ {
	case config.TypeProxyHTTP:
		return factory.httpProxy()
	case config.TypeProxySocks5:
		return factory.socks5Proxy()
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedProxyType, factory.proxy.Typ)
	}
}

func (factory *transportFactory) direct() (*http.Transport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}

	transport := base.Clone()
	transport.Proxy = nil

	return transport, nil
}

func (factory *transportFactory) httpProxy() (*http.Transport, error) {
	proxyURL, err := url.Parse(factory.proxy.Addr)
	if err != nil {
		return nil, fmt.Errorf("parse proxy addr: %w", domain.ErrInvalidProxyAddr)
	}

	if factory.proxy.Login != "" {
		proxyURL.User = url.UserPassword(factory.proxy.Login, factory.proxy.Passw)
	}

	return &http.Transport{Proxy: http.ProxyURL(proxyURL)}, nil
}

func (factory *transportFactory) socks5Proxy() (*http.Transport, error) {
	var auth *proxy.Auth
	if factory.proxy.Login != "" {
		auth = &proxy.Auth{User: factory.proxy.Login, Password: factory.proxy.Passw}
	}

	dialer, err := proxy.SOCKS5("tcp", factory.proxy.Addr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("setup socks5 proxy: %w", err)
	}

	return &http.Transport{DialContext: dialer.(proxy.ContextDialer).DialContext}, nil
}

func (t *modelLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.retryStatus.Set(0, 0)

		return nil, err
	}
	if resp.Body == nil {
		t.retryStatus.Set(resp.StatusCode, 0)

		return resp, nil
	}

	originalBody := resp.Body
	body, readErr := io.ReadAll(originalBody)
	closeErr := originalBody.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	t.retryStatus.Set(resp.StatusCode, retry.ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now()))

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

	t.log.Info("LLM HTTP response received",
		zap.String("actor", "llm"),
		zap.String("phase", "http_response"),
		zap.String("model", payload.Model))
}
