package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yvv4git/git-commit-gen/internal/application"
)

func SetupGenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "gen",
		Short:        "Generate a commit message for the current branch",
		SilenceUsage: true,
		Example: `  # Generate a commit message with the default config
  git-commit-gen gen

  # Generate a commit message and print the result
  git-commit-gen gen --visual
  git-commit-gen gen -v

  # Generate a commit message with another LLM provider
  git-commit-gen gen --provider secondary
  git-commit-gen gen -p secondary

  # Generate a commit message with an explicit config
  git-commit-gen gen --config /path/to/config.toml
  git-commit-gen gen -c /path/to/config.toml`,
		RunE: runGen,
	}

	cmd.Flags().BoolP("visual", "v", false, "print generated commit message")
	cmd.Flags().StringP("provider", "p", "", "LLM provider name")

	return cmd
}

type genFlags struct {
	visual   bool
	provider string
}

func (flags *genFlags) parse(cmd *cobra.Command) error {
	visual, err := cmd.Flags().GetBool("visual")
	if err != nil {
		return fmt.Errorf("get visual flag: %w", err)
	}

	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		return fmt.Errorf("get provider flag: %w", err)
	}

	flags.visual = visual
	flags.provider = provider

	return nil
}

func runGen(cmd *cobra.Command, _ []string) error {
	var flags genFlags
	if err := flags.parse(cmd); err != nil {
		return err
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("get config flag: %w", err)
	}

	app, err := application.NewGenApplication(cmd.Context(), application.GenOptions{
		ConfigPath: configPath,
		Visual:     flags.visual,
		Provider:   flags.provider,
	})
	if err != nil {
		return err
	}
	defer app.Shutdown()

	return app.Run()
}
