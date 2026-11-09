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
	ProviderCheckResult struct {
		Value string
	}
)

type LLM interface {
	GenCommitDescription(ctx context.Context) (*GenCommitDescriptionResult, error)
}

type ProviderChecker interface {
	CheckProvider(ctx context.Context) (*ProviderCheckResult, error)
}
