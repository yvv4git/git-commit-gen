package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/config/generator"
)

func TestSelectLLMProvider(t *testing.T) {
	providers := config.LLMs{{Name: "primary"}, {Name: "secondary"}}

	tests := []struct {
		name      string
		providers config.LLMs
		provider  string
		want      config.LLM
		wantErr   bool
	}{
		{name: "empty list", wantErr: true},
		{name: "default provider", providers: providers, want: config.LLM{Name: "primary"}},
		{name: "provider by name", providers: providers, provider: "secondary", want: config.LLM{Name: "secondary"}},
		{name: "unknown provider", providers: providers, provider: "missing", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := &GenApplication{cfg: generator.Config{LLMs: tt.providers}}

			got, err := application.selectLLMProvider(tt.provider)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
