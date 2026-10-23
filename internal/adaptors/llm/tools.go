package llm

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

const (
	toolGetCurrentBranch  = "GetCurrentBranch"
	toolLoadDiff          = "LoadDiff"
	toolUpdateFirstCommit = "UpdateFirstCommit"
)

type updateFirstCommitArgs struct {
	CommitMessage string `json:"commit_message"`
}

func Tools() []llms.Tool {
	return []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        toolGetCurrentBranch,
				Description: "Return the name of the current git branch.",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        toolLoadDiff,
				Description: "Return the diff between the merge base and the current branch.",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        toolUpdateFirstCommit,
				Description: "Replace the message of the first commit on the current branch.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"commit_message": {
							"type": "string",
							"description": "The commit message to apply."
						}
					},
					"required": ["commit_message"]
				}`),
			},
		},
	}
}
