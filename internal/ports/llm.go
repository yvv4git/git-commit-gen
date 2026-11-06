package ports

import "context"

type (
	GenCommitDescriptionResult struct {
		Value            string
		BackupRef        string
		PromptTokens     int
		CompletionTokens int
		TotalTokens      int
		Requests         int
	}
)

type LLM interface {
	GenCommitDescription(ctx context.Context) (*GenCommitDescriptionResult, error)
}
