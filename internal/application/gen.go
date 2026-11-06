package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/git"
	"github.com/yvv4git/git-commit-gen/internal/adaptors/llm"
	"github.com/yvv4git/git-commit-gen/internal/config"
	"github.com/yvv4git/git-commit-gen/internal/config/generator"
	generatoruc "github.com/yvv4git/git-commit-gen/internal/core/generator"
	"github.com/yvv4git/git-commit-gen/internal/infra"
	"github.com/yvv4git/git-commit-gen/internal/ports"
	"github.com/yvv4git/git-commit-gen/pkg/terminal"
)

type GenOptions struct {
	ConfigPath string
	Visual     bool
	Provider   string
}

type GenApplication struct {
	ctx       context.Context
	stop      context.CancelFunc
	log       *zap.Logger
	cfg       generator.Config
	opts      GenOptions
	generator *generatoruc.Generator
}

func NewGenApplication(parent context.Context, opts GenOptions) (*GenApplication, error) {
	var cfg generator.Config
	if err := generator.Load(opts.ConfigPath, &cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)

	log, err := infra.NewWithLogLevel(cfg.Log.Level)
	if err != nil {
		stop()

		return nil, fmt.Errorf("setup logger: %w", err)
	}

	app := &GenApplication{
		ctx:  ctx,
		stop: stop,
		log:  log,
		cfg:  cfg,
		opts: opts,
	}

	app.log.Info("Starting",
		zap.String("command", "gen"),
		zap.String("config_path", opts.ConfigPath),
		zap.String("log_level", cfg.Log.Level))

	generatorEntity, err := app.newGenerator()
	if err != nil {
		app.Shutdown()

		return nil, err
	}

	app.generator = generatorEntity

	return app, nil
}

func (a *GenApplication) Run() error {
	result, err := a.generator.Gen(a.ctx)
	if err != nil {
		return fmt.Errorf("generate commit: %w", err)
	}

	if a.opts.Visual {
		fmt.Print(terminal.FormatResult(terminal.Result{
			CommitMessage:    result.Value,
			BackupRef:        result.BackupRef,
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.TotalTokens,
			Requests:         result.Requests,
		}))
	}

	return nil
}

func (a *GenApplication) Shutdown() {
	a.stop()

	_ = a.log.Sync()
}

func (a *GenApplication) newGenerator() (*generatoruc.Generator, error) {
	llmCfg, err := a.selectLLMProvider(a.opts.Provider)
	if err != nil {
		return nil, err
	}

	rulesFile, err := fs.NewFS().ReadFile(a.ctx, &ports.ReadFileParams{FilePath: a.cfg.Gen.RulesFile})
	if err != nil {
		return nil, fmt.Errorf("read rules file: %w", err)
	}

	llmLog := a.log.Named("LLM")
	llmClient, retryStatus, err := infra.SetupLLMClient(llmCfg, llmLog)
	if err != nil {
		return nil, fmt.Errorf("setup llm client: %w", err)
	}

	llmLog.Info("LLM provider", zap.String("name", llmCfg.Name))

	gitClient := git.New()
	llmAdapter := llm.NewOpenAI(llmLog, llmClient, rulesFile.Value, gitClient, a.llmOptions(llmCfg, retryStatus)...)

	return generatoruc.NewGenerator(a.cfg.Gen.BaseBranch, gitClient, llmAdapter), nil
}

func (a *GenApplication) llmOptions(llmCfg config.LLM, retryStatus llm.RetryStatusProvider) []llm.Option {
	opts := []llm.Option{
		llm.WithBaseBranch(a.cfg.Gen.BaseBranch),
		llm.WithRetry(llm.RetrySettings{
			Retries:      llmCfg.Retry.Retries,
			InitInterval: llmCfg.Retry.InitInterval,
			MaxInterval:  llmCfg.Retry.MaxInterval,
		}),
		llm.WithRetryStatus(retryStatus),
	}

	return append(opts, a.limitOptions()...)
}

func (a *GenApplication) limitOptions() []llm.Option {
	var opts []llm.Option

	if a.cfg.Gen.MaxDiffBytes > 0 {
		opts = append(opts, llm.WithMaxDiffBytes(a.cfg.Gen.MaxDiffBytes))
	}
	if a.cfg.Gen.MaxRulesBytes > 0 {
		opts = append(opts, llm.WithMaxRulesBytes(a.cfg.Gen.MaxRulesBytes))
	}
	if a.cfg.Gen.MaxContextBytes > 0 {
		opts = append(opts, llm.WithMaxContextBytes(a.cfg.Gen.MaxContextBytes))
	}
	if a.cfg.Gen.MaxAgentTurns > 0 {
		opts = append(opts, llm.WithMaxAgentTurns(a.cfg.Gen.MaxAgentTurns))
	}
	if a.cfg.Gen.MaxToolCallsPerTurn > 0 {
		opts = append(opts, llm.WithMaxToolCallsPerTurn(a.cfg.Gen.MaxToolCallsPerTurn))
	}

	return opts
}

func (a *GenApplication) selectLLMProvider(name string) (config.LLM, error) {
	if len(a.cfg.LLMs) == 0 {
		return config.LLM{}, errors.New("no LLM providers configured")
	}

	if name == "" {
		return a.cfg.LLMs[0], nil
	}

	for _, llmCfg := range a.cfg.LLMs {
		if llmCfg.Name == name {
			return llmCfg, nil
		}
	}

	return config.LLM{}, fmt.Errorf("LLM provider %q not found", name)
}
