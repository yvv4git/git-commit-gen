package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/yvv4git/git-commit-gen/cmd/generator/commands"
)

func main() {
	rootCommand := &cobra.Command{
		Use:   "crawler",
		Short: "Root command for Crawler application",
	}
	rootCommand.PersistentFlags().StringP("config", "c", "config.toml", "Path to config file")

	rootCommand.AddCommand(commands.SetupGenCommand())

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
