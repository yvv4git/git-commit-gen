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

type (
	WriteFileParams struct {
		FilePath string
		Content  string
	}
	WriteFileResult struct{}
)

type FileReader interface {
	ReadFile(ctx context.Context, params *ReadFileParams) (*ReadFileResult, error)
}

type FileWriter interface {
	WriteFile(ctx context.Context, params *WriteFileParams) error
}

type ConfigPath interface {
	DefaultConfigDir(ctx context.Context) (string, error)
}
