package ports

import "context"

type (
	GenCommitDescriptionParams struct {
		Diff string
	}
	GenCommitDescriptionResult struct {
		Value string
	}
)

type LLM interface {
	GenCommitDescription(ctx context.Context, params *GenCommitDescriptionParams) (*GenCommitDescriptionResult, error)
}
