package llm

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

func Tools() []llms.Tool {
	return []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "get_current_branch",
				Description: "Get the name of the current git branch. Use this when you need to include the branch name in the commit message.",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
	}
}
