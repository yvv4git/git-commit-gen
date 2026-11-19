package setup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type fakeFS struct {
	dir   string
	files map[string]string
}

func (f *fakeFS) DefaultConfigDir(context.Context) (string, error) {
	return f.dir, nil
}

func (f *fakeFS) MkdirAll(context.Context, *ports.MkdirAllParams) error {
	return nil
}

func (f *fakeFS) FileExists(_ context.Context, params *ports.FileExistsParams) (bool, error) {
	_, ok := f.files[params.FilePath]

	return ok, nil
}

func (f *fakeFS) WriteFile(_ context.Context, params *ports.WriteFileParams) error {
	f.files[params.FilePath] = params.Content

	return nil
}

func TestSetupRun(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	rulesPath := filepath.Join(dir, "rules.md")
	defaultFiles := map[string]string{
		configPath: "default config",
		rulesPath:  "default rules",
	}

	tests := []struct {
		name        string
		existing    map[string]string
		force       bool
		wantFiles   map[string]string
		wantSkipped []string
	}{
		{
			name:      "create both files",
			wantFiles: defaultFiles,
		},
		{
			name: "skip both files",
			existing: map[string]string{
				configPath: "custom config",
				rulesPath:  "custom rules",
			},
			wantFiles: map[string]string{
				configPath: "custom config",
				rulesPath:  "custom rules",
			},
			wantSkipped: []string{configPath, rulesPath},
		},
		{
			name: "skip config and create rules",
			existing: map[string]string{
				configPath: "custom config",
			},
			wantFiles: map[string]string{
				configPath: "custom config",
				rulesPath:  "default rules",
			},
			wantSkipped: []string{configPath},
		},
		{
			name: "skip rules and create config",
			existing: map[string]string{
				rulesPath: "custom rules",
			},
			wantFiles: map[string]string{
				configPath: "default config",
				rulesPath:  "custom rules",
			},
			wantSkipped: []string{rulesPath},
		},
		{
			name: "force overwrite both files",
			existing: map[string]string{
				configPath: "custom config",
				rulesPath:  "custom rules",
			},
			force:     true,
			wantFiles: defaultFiles,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{}
			for path, content := range tt.existing {
				files[path] = content
			}
			fileSystem := &fakeFS{dir: dir, files: files}

			result, err := NewSetup(fileSystem, fileSystem).Run(context.Background(), &SetupParams{
				ConfigContent: "default config",
				RulesContent:  "default rules",
				Force:         tt.force,
			})
			require.NoError(t, err)
			require.Equal(t, tt.wantFiles, files)
			require.Equal(t, tt.wantSkipped, result.Skipped)
		})
	}
}
