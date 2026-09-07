package fs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

const (
	ConfigDirName    = ".config"
	ConfigAppDirName = "git_commit_gen"
	ConfigFileName   = "config.toml"
)

type FS struct{}

func NewFS() *FS {
	return &FS{}
}

func (f *FS) ReadFile(ctx context.Context, params *ports.ReadFileParams) (*ports.ReadFileResult, error) {
	path := params.FilePath
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			dir, err := f.DefaultConfigDir(ctx)
			if err != nil {
				return nil, err
			}
			path = filepath.Join(dir, ConfigFileName)
		} else {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
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

	return filepath.Join(home, ConfigDirName, ConfigAppDirName), nil
}
