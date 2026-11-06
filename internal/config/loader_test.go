package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDotEnvMissing(t *testing.T) {
	t.Chdir(t.TempDir())

	values, err := LoadDotEnv()
	require.NoError(t, err)
	require.Empty(t, values)
}

func TestLoadDotEnvDoesNotSetProcessEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_GEN_DOTENV_MARKER", "")
	require.NoError(t, os.WriteFile(DotEnvFileName, []byte("GIT_GEN_DOTENV_MARKER=secret\n"), 0o600))

	values, err := LoadDotEnv()
	require.NoError(t, err)
	require.Equal(t, "secret", values["GIT_GEN_DOTENV_MARKER"])
	require.Empty(t, os.Getenv("GIT_GEN_DOTENV_MARKER"))
}
