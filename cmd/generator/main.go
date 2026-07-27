package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yvv4git/git-commit-gen/cmd/generator/commands"
)

func main() {
	rootCommand := &cobra.Command{
		Use:   "crawler",
		Short: "Root command for Crawler application",
	}
	rootCommand.PersistentFlags().StringP("config", "c", defaultConfigPath(), "Path to config file")

	rootCommand.AddCommand(commands.GenCommand())
	rootCommand.AddCommand(commands.SetupCommand())

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}

	return filepath.Join(home, ".config", "git_commit_gen", "config.toml")
}
