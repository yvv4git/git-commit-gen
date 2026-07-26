package fs

import (
	"context"
	"os"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type FS struct{}

func NewFS() *FS {
	return &FS{}
}

func (f *FS) ReadFile(_ context.Context, params *ports.ReadFileParams) (*ports.ReadFileResult, error) {
	data, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, err
	}

	return &ports.ReadFileResult{Value: string(data)}, nil
}
