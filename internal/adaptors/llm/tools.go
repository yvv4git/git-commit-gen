package llm

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

const (
	toolGetCurrentBranch  = "GetCurrentBranch"
	toolLoadDiff          = "LoadDiff"
	toolUpdateFirstCommit = "UpdateFirstCommit"
	toolVerifyFirstCommit = "VerifyFirstCommit"
)

type updateFirstCommitArgs struct {
	CommitMessage string `json:"commit_message"`
}

// +LLM06:2025 Excessive Agency
// +ASI02:2026 Tool Misuse and Exploitation
// +ASI10:2026 Rogue Agents
// +T2 Tool Misuse
// -LLM08:2025 Vector and Embedding Weaknesses
// -ASI07:2026 Insecure Inter-Agent Communication
// -T12 Agent Communication Poisoning
// -T13 Rogue Agents in Multi-Agent Systems
// -T14 Human Attacks on Multi-Agent Systems
// -T16 Insecure Inter-Agent Protocol Abuse
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
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        toolVerifyFirstCommit,
				Description: "Verify that the first commit on the current branch has the generated commit message.",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
	}
}
