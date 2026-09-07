package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yvv4git/git-commit-gen/cmd"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
)

//go:embed config.example.toml
var defaultConfigContent string

//go:embed rules.example.md
var defaultRulesContent string

func main() {
	rootCommand := &cobra.Command{
		Use:   "crawler",
		Short: "Root command for Crawler application",
	}
	rootCommand.PersistentFlags().StringP("config", "c", defaultConfigPath(), "Path to config file")

	rootCommand.AddCommand(commands.GenCommand())
	rootCommand.AddCommand(commands.SetupCommand(defaultConfigContent, defaultRulesContent))

	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}

	return filepath.Join(home, fs.ConfigDirName, fs.ConfigAppDirName, fs.ConfigFileName)
}
