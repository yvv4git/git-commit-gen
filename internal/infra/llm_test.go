package infra

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type stubTransport struct {
	body string
	err  error
}

func (s *stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(s.body)),
	}, nil
}

func TestModelLoggingTransportLogsModel(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	transport := &modelLoggingTransport{
		base: &stubTransport{body: `{"model":"poolside/laguna-s-2.1:free"}`},
		log:  zap.New(core),
	}
	req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, `{"model":"poolside/laguna-s-2.1:free"}`, string(body))
	entries := logs.FilterMessage("LLM model").All()
	require.Len(t, entries, 1)
	assert.Equal(t, "poolside/laguna-s-2.1:free", entries[0].ContextMap()["model"])
}

func TestModelLoggingTransportSkipsBodyWithoutModel(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "no model field", body: `{"choices":[]}`},
		{name: "empty model", body: `{"model":""}`},
		{name: "not json", body: `not json`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			transport := &modelLoggingTransport{
				base: &stubTransport{body: tt.body},
				log:  zap.New(core),
			}
			req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1", nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, tt.body, string(body))
			assert.Empty(t, logs.FilterMessage("LLM model").All())
		})
	}
}

func TestModelLoggingTransportReturnsBaseError(t *testing.T) {
	wantErr := errors.New("network down")
	transport := &modelLoggingTransport{
		base: &stubTransport{err: wantErr},
		log:  zap.NewNop(),
	}
	req, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, resp)
}
