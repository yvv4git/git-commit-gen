package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yvv4git/git-commit-gen/internal/application"
)

func SetupCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "check",
		Short:        "Check that the configured LLM provider is working",
		SilenceUsage: true,
		Example: `  # Check the default provider
  git-commit-gen check

  # Check a specific provider
  git-commit-gen check --provider ollama
  git-commit-gen check -p ollama

  # Check with an explicit config
  git-commit-gen check --config /path/to/config.toml
  git-commit-gen check -c /path/to/config.toml`,
		RunE: runCheck,
	}

	cmd.Flags().StringP("provider", "p", "", "LLM provider name")

	return cmd
}

func runCheck(cmd *cobra.Command, _ []string) error {
	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		return fmt.Errorf("get provider flag: %w", err)
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("get config flag: %w", err)
	}

	app, err := application.NewCheckApplication(cmd.Context(), application.CheckOptions{
		ConfigPath: configPath,
		Provider:   provider,
	})
	if err != nil {
		return err
	}
	defer app.Shutdown()

	return app.Run()
}
