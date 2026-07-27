package commands

import (
	"github.com/spf13/cobra"
	"github.com/yvv4git/git-commit-gen"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
	"github.com/yvv4git/git-commit-gen/internal/core"
)

func SetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Create default config and rules files",
		Long: `Create default configuration and rules files in the OS-specific config directory.

Example:
  git-commit-gen setup`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var f fs.FS
			setup := core.NewSetup(&f, &f)

			return setup.Run(cmd.Context(), &core.SetupParams{
				ConfigContent: defaults.ConfigContent,
				RulesContent:  defaults.RulesContent,
			})
		},
	}
}
