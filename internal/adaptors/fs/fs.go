package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

const (
	ConfigDirName    = ".config"
	ConfigAppDirName = "git_commit_gen"
	ConfigFileName   = "config.toml"
	RulesFileName    = "rules.md"
	fileMode         = 0o600
	dirMode          = 0o700
)

type FS struct{}

func NewFS() *FS {
	return &FS{}
}

func (f *FS) ReadFile(ctx context.Context, params *ports.ReadFileParams) (*ports.ReadFileResult, error) {
	path, err := f.resolveReadPath(ctx, params.FilePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", domain.ErrRulesNotFound, path)
		}

		return nil, err
	}

	return &ports.ReadFileResult{Value: string(data)}, nil
}

func (f *FS) resolveReadPath(ctx context.Context, path string) (string, error) {
	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	dir, err := f.DefaultConfigDir(ctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, RulesFileName), nil
}

func (f *FS) WriteFile(_ context.Context, params *ports.WriteFileParams) error {
	if err := os.WriteFile(params.FilePath, []byte(params.Content), fileMode); err != nil {
		return err
	}

	return os.Chmod(params.FilePath, fileMode)
}

func (f *FS) MkdirAll(_ context.Context, params *ports.MkdirAllParams) error {
	if err := os.MkdirAll(params.DirPath, dirMode); err != nil {
		return err
	}

	return os.Chmod(params.DirPath, dirMode)
}

func (f *FS) DefaultConfigDir(_ context.Context) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ConfigDirName, ConfigAppDirName), nil
}
