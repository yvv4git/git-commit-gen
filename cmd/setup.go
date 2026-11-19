package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yvv4git/git-commit-gen/internal/application"
)

func SetupCommand(configContent, rulesContent string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Create default config and rules files",
		Long: `Create default configuration and rules files in the OS-specific config directory.

Example:
  git-commit-gen setup
  git-commit-gen setup --force
  git-commit-gen setup -f`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return fmt.Errorf("get force flag: %w", err)
			}

			app := application.NewSetupApplication(cmd.Context(), application.SetupOptions{
				ConfigContent: configContent,
				RulesContent:  rulesContent,
				Force:         force,
			})
			defer app.Shutdown()

			result, err := app.Run()
			if err != nil {
				return err
			}

			for _, path := range result.Skipped {
				cmd.PrintErrf("warning: %s already exists, skipping (use --force to overwrite)\n", path)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "overwrite existing config and rules files")

	return cmd
}
