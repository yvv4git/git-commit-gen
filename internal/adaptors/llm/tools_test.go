package llm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireJSON(t *testing.T, want string, value any) {
	t.Helper()

	raw, ok := value.(json.RawMessage)
	require.True(t, ok)
	assert.JSONEq(t, want, string(raw))
}

func TestTools(t *testing.T) {
	tools := Tools()

	require.Len(t, tools, 4)

	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		require.NotNil(t, tool.Function)
		names = append(names, tool.Function.Name)
	}

	assert.Equal(t, []string{toolGetCurrentBranch, toolLoadDiff, toolUpdateFirstCommit, toolVerifyFirstCommit}, names)
	requireJSON(t, `{"type": "object", "properties": {}}`, tools[0].Function.Parameters)
	requireJSON(t, `{"type": "object", "properties": {}}`, tools[1].Function.Parameters)
	requireJSON(t, `{
		"type": "object",
		"properties": {
			"commit_message": {
				"type": "string",
				"description": "The commit message to apply."
			}
		},
		"required": ["commit_message"]
	}`, tools[2].Function.Parameters)
	requireJSON(t, `{"type": "object", "properties": {}}`, tools[3].Function.Parameters)
}
