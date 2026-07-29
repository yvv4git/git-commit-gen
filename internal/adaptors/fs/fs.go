package fs

import (
	"context"
	"os"
	"path/filepath"

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

func (f *FS) WriteFile(_ context.Context, params *ports.WriteFileParams) error {
	return os.WriteFile(params.FilePath, []byte(params.Content), 0644)
}

func (f *FS) MkdirAll(_ context.Context, params *ports.MkdirAllParams) error {
	return os.MkdirAll(params.DirPath, 0755)
}

func (f *FS) DefaultConfigDir(_ context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "git_commit_gen"), nil
}
