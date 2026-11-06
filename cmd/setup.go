package commands

import (
	"github.com/spf13/cobra"

	"github.com/yvv4git/git-commit-gen/internal/application"
)

func SetupCommand(configContent, rulesContent string) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Create default config and rules files",
		Long: `Create default configuration and rules files in the OS-specific config directory.

Example:
  git-commit-gen setup`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			app := application.NewSetupApplication(cmd.Context(), application.SetupOptions{
				ConfigContent: configContent,
				RulesContent:  rulesContent,
			})
			defer app.Shutdown()

			return app.Run()
		},
	}
}
