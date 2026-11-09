package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestCheckProviderReturnsReply(t *testing.T) {
	transport := &fakeTransport{responses: []*llms.ContentResponse{
		newTextResponse("  I am Qwen.  "),
	}}
	adapter := newTestAdapter(t, &fakeGit{}, transport, "")

	result, err := adapter.CheckProvider(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "I am Qwen.", result.Value)
	assert.Equal(t, 1, transport.calls)
}

func TestCheckProviderEmptyResponse(t *testing.T) {
	transport := &fakeTransport{responses: []*llms.ContentResponse{{}}}
	adapter := newTestAdapter(t, &fakeGit{}, transport, "")

	_, err := adapter.CheckProvider(context.Background())

	require.Error(t, err)
}
