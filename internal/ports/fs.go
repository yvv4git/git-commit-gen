package ports

import "context"

type (
	ReadFileParams struct {
		FilePath string
	}
	ReadFileResult struct {
		Value string
	}
)

type FileReader interface {
	ReadFile(ctx context.Context, params *ReadFileParams) (*ReadFileResult, error)
}
