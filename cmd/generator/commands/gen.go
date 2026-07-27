package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/git"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/llm"
	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/config/generator"
	"github.com/yvv4git/git-commit-gen/internal/core"
	"github.com/yvv4git/git-commit-gen/internal/infra"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

func GenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Run generate commmit",
		Long: `Start commit generate.

Example:
  git-commit-gen --config /path/to/config.toml`,
		RunE: gen,
	}

	cmd.Flags().BoolP("visual", "v", false, "print generated commit message")

	return cmd
}

func gen(cmd *cobra.Command, args []string) error {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	visual, err := cmd.Flags().GetBool("visual")
	if err != nil {
		return fmt.Errorf("get visual flag: %w", err)
	}

	var cfg generator.Config
	if err := config.Load(cfgPath, &cfg); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log, err := infra.NewWithLogLevel(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("setup logger: %w", err)
	}

	gitClient := git.New()

	llmClient, err := infra.SetupOpenAIClient(cfg.LLM)
	if err != nil {
		return fmt.Errorf("setup llm client: %w", err)
	}

	fsClient := fs.NewFS()
	fileRes, err := fsClient.ReadFile(ctx, &ports.ReadFileParams{FilePath: cfg.Gen.RulesFile})
	if err != nil {
		return fmt.Errorf("read rules file: %w", err)
	}

	llmAdapter := llm.NewOpenAI(log, llmClient, fileRes.Value, gitClient)

	gen := core.NewGenerator(cfg.Gen.BaseBranch, visual, gitClient, llmAdapter)

	return gen.Gen(ctx)
}
