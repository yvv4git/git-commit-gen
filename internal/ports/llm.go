package ports

import "context"

type (
	GenCommitDescriptionResult struct {
		Value string
	}
)

type LLM interface {
	GenCommitDescription(ctx context.Context) (*GenCommitDescriptionResult, error)
}
