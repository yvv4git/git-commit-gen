package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

func TestReadFileFromWorkingDirectory(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, RulesFileName), []byte("cwd rules"), 0600))

	result, err := NewFS().ReadFile(context.Background(), &ports.ReadFileParams{FilePath: RulesFileName})
	require.NoError(t, err)
	require.Equal(t, "cwd rules", result.Value)
}

func TestReadFileFallsBackToDefaultRules(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())

	defaultDir := filepath.Join(home, ConfigDirName, ConfigAppDirName)
	require.NoError(t, os.MkdirAll(defaultDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(defaultDir, RulesFileName), []byte("default rules"), 0600))

	result, err := NewFS().ReadFile(context.Background(), &ports.ReadFileParams{FilePath: RulesFileName})
	require.NoError(t, err)
	require.Equal(t, "default rules", result.Value)
}

func TestReadFileReturnsErrorWhenNoRules(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())

	result, err := NewFS().ReadFile(context.Background(), &ports.ReadFileParams{FilePath: RulesFileName})
	require.ErrorIs(t, err, domain.ErrRulesNotFound)
	require.Nil(t, result)
}

func TestReadFileDoesNotFallBackToConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())

	defaultDir := filepath.Join(home, ConfigDirName, ConfigAppDirName)
	require.NoError(t, os.MkdirAll(defaultDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(defaultDir, ConfigFileName), []byte("token = secret"), 0600))

	result, err := NewFS().ReadFile(context.Background(), &ports.ReadFileParams{FilePath: RulesFileName})
	require.ErrorIs(t, err, domain.ErrRulesNotFound)
	require.Nil(t, result)
}
