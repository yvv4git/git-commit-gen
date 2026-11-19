package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	fsadapter "github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
)

func TestSetupCommandSkipsExistingFileWithWarning(t *testing.T) {
	dir := prepareSetupHome(t)
	configPath := filepath.Join(dir, fsadapter.ConfigFileName)
	rulesPath := filepath.Join(dir, fsadapter.RulesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte("custom config"), 0600))

	cmd := SetupCommand("default config", "default rules")
	cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	require.NoError(t, cmd.Execute())
	require.Contains(t, stderr.String(), "warning:")
	require.Contains(t, stderr.String(), fsadapter.ConfigFileName)
	require.Contains(t, stderr.String(), "already exists")

	config, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Equal(t, "custom config", string(config))

	rules, err := os.ReadFile(rulesPath)
	require.NoError(t, err)
	require.Equal(t, "default rules", string(rules))
}

func TestSetupCommandForceOverwritesExistingFiles(t *testing.T) {
	for _, flag := range []string{"-f", "--force"} {
		t.Run(flag, func(t *testing.T) {
			dir := prepareSetupHome(t)
			configPath := filepath.Join(dir, fsadapter.ConfigFileName)
			rulesPath := filepath.Join(dir, fsadapter.RulesFileName)
			require.NoError(t, os.WriteFile(configPath, []byte("custom config"), 0600))
			require.NoError(t, os.WriteFile(rulesPath, []byte("custom rules"), 0600))

			cmd := SetupCommand("default config", "default rules")
			cmd.SetArgs([]string{flag})

			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			require.NoError(t, cmd.Execute())
			require.Empty(t, stderr.String())

			config, err := os.ReadFile(configPath)
			require.NoError(t, err)
			require.Equal(t, "default config", string(config))

			rules, err := os.ReadFile(rulesPath)
			require.NoError(t, err)
			require.Equal(t, "default rules", string(rules))
		})
	}
}

func prepareSetupHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, fsadapter.ConfigDirName, fsadapter.ConfigAppDirName)
	require.NoError(t, os.MkdirAll(dir, 0700))

	return dir
}
