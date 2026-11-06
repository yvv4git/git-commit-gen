package infra

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/domain"
)

func TestNewHTTPClientIgnoresEnvProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example:8080")
	t.Setenv("HTTPS_PROXY", "http://proxy.example:8080")

	client, _, err := newHTTPClient(config.Proxy{}, 0, zap.NewNop())
	require.NoError(t, err)

	loggingTransport, ok := client.Transport.(*modelLoggingTransport)
	require.True(t, ok)
	transport, ok := loggingTransport.base.(*http.Transport)
	require.True(t, ok)
	require.Nil(t, transport.Proxy)
}

func TestNewHTTPClientRejectsInvalidProxyAddr(t *testing.T) {
	proxy := config.Proxy{Enable: true, Typ: config.TypeProxyHTTP, Addr: "http://user:secret@[::1"}

	_, _, err := newHTTPClient(proxy, 0, zap.NewNop())
	require.ErrorIs(t, err, domain.ErrInvalidProxyAddr)
	require.NotContains(t, err.Error(), "secret")
}

func TestSetupLLMClientRejectsInsecureHTTP(t *testing.T) {
	cfg := config.LLM{OpenAI: config.OpenAI{API: "http://api.example.com/v1", Token: "token", Model: "model"}}

	_, _, err := SetupLLMClient(cfg, zap.NewNop())

	require.ErrorIs(t, err, domain.ErrInsecureAPIURL)
}

func TestSetupLLMClientAllowsLocalhostHTTP(t *testing.T) {
	tests := []string{
		"http://localhost:8080/v1",
		"http://127.0.0.1:8080/v1",
		"http://[::1]:8080/v1",
	}

	for _, api := range tests {
		t.Run(api, func(t *testing.T) {
			cfg := config.LLM{OpenAI: config.OpenAI{API: api, Token: "token", Model: "model"}}

			_, _, err := SetupLLMClient(cfg, zap.NewNop())

			require.NoError(t, err)
		})
	}
}

func TestSetupLLMClientAllowsInsecureHTTPByFlag(t *testing.T) {
	cfg := config.LLM{OpenAI: config.OpenAI{
		API:               "http://api.example.com/v1",
		Token:             "token",
		Model:             "model",
		AllowInsecureHTTP: true,
	}}

	_, _, err := SetupLLMClient(cfg, zap.NewNop())

	require.NoError(t, err)
}

func TestSetupLLMClientAllowsHTTPS(t *testing.T) {
	cfg := config.LLM{OpenAI: config.OpenAI{API: "https://api.example.com/v1", Token: "token", Model: "model"}}

	_, _, err := SetupLLMClient(cfg, zap.NewNop())

	require.NoError(t, err)
}

func TestSetupLLMClientRejectsInvalidAPIURL(t *testing.T) {
	cfg := config.LLM{OpenAI: config.OpenAI{API: "://bad", Token: "token", Model: "model"}}

	_, _, err := SetupLLMClient(cfg, zap.NewNop())

	require.ErrorIs(t, err, domain.ErrInvalidAPIURL)
}

func TestSetupLLMClientRejectsUnsupportedScheme(t *testing.T) {
	cfg := config.LLM{OpenAI: config.OpenAI{API: "ftp://api.example.com/v1", Token: "token", Model: "model"}}

	_, _, err := SetupLLMClient(cfg, zap.NewNop())

	require.ErrorIs(t, err, domain.ErrInvalidAPIURL)
}

func TestNewHTTPClientSetsDefaultTimeout(t *testing.T) {
	client, _, err := newHTTPClient(config.Proxy{}, 0, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, defaultRequestTimeout, client.Timeout)
}

func TestNewHTTPClientUsesConfiguredTimeout(t *testing.T) {
	const timeout = 5 * time.Second

	client, _, err := newHTTPClient(config.Proxy{}, timeout, zap.NewNop())
	require.NoError(t, err)
	require.Equal(t, timeout, client.Timeout)
}
