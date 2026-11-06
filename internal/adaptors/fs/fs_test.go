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

func TestWriteFileSetsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)

	err := NewFS().WriteFile(context.Background(), &ports.WriteFileParams{FilePath: path, Content: "test"})
	require.NoError(t, err)
	requirePermissions(t, path, 0o600)
}

func TestWriteFileFixesExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	err := NewFS().WriteFile(context.Background(), &ports.WriteFileParams{FilePath: path, Content: "new"})
	require.NoError(t, err)
	requirePermissions(t, path, 0o600)
}

func TestMkdirAllSetsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigAppDirName)

	err := NewFS().MkdirAll(context.Background(), &ports.MkdirAllParams{DirPath: path})
	require.NoError(t, err)
	requirePermissions(t, path, 0o700)
}

func TestMkdirAllFixesExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigAppDirName)
	require.NoError(t, os.Mkdir(path, 0o755))

	err := NewFS().MkdirAll(context.Background(), &ports.MkdirAllParams{DirPath: path})
	require.NoError(t, err)
	requirePermissions(t, path, 0o700)
}

func requirePermissions(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, mode, info.Mode().Perm())
}
