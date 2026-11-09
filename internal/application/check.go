package application

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/yvv4git/git-commit-gen/internal/adaptors/llm"
	"github.com/yvv4git/git-commit-gen/internal/config/generator"
	checkeruc "github.com/yvv4git/git-commit-gen/internal/core/checker"
	"github.com/yvv4git/git-commit-gen/internal/infra"
	"github.com/yvv4git/git-commit-gen/pkg/terminal"
)

type CheckOptions struct {
	ConfigPath string
	Provider   string
}

type CheckApplication struct {
	ctx      context.Context
	stop     context.CancelFunc
	log      *zap.Logger
	cfg      generator.Config
	opts     CheckOptions
	checker  *checkeruc.Checker
	provider string
}

func NewCheckApplication(parent context.Context, opts CheckOptions) (*CheckApplication, error) {
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

	app := &CheckApplication{
		ctx:  ctx,
		stop: stop,
		log:  log,
		cfg:  cfg,
		opts: opts,
	}

	app.log.Info("Starting",
		zap.String("command", "check"),
		zap.String("config_path", opts.ConfigPath),
		zap.String("log_level", cfg.Log.Level))

	checkerEntity, providerName, err := app.newChecker()
	if err != nil {
		app.Shutdown()

		return nil, err
	}

	app.checker = checkerEntity
	app.provider = providerName

	return app, nil
}

func (a *CheckApplication) Run() error {
	result, err := a.checker.Check(a.ctx)
	if err != nil {
		return fmt.Errorf("check provider: %w", err)
	}

	fmt.Print(terminal.FormatCheck(a.provider, result.Value))

	return nil
}

func (a *CheckApplication) Shutdown() {
	a.stop()

	_ = a.log.Sync()
}

func (a *CheckApplication) newChecker() (*checkeruc.Checker, string, error) {
	llmCfg, err := selectLLMProvider(a.cfg, a.opts.Provider)
	if err != nil {
		return nil, "", err
	}

	llmLog := a.log.Named("LLM")
	llmClient, retryStatus, err := infra.SetupLLMClient(llmCfg, llmLog)
	if err != nil {
		return nil, "", fmt.Errorf("setup llm client: %w", err)
	}

	llmLog.Info("LLM provider", zap.String("name", llmCfg.Name))

	opts := []llm.Option{
		llm.WithRetry(llm.RetrySettings{
			Retries:      llmCfg.Retry.Retries,
			InitInterval: llmCfg.Retry.InitInterval,
			MaxInterval:  llmCfg.Retry.MaxInterval,
		}),
		llm.WithRetryStatus(retryStatus),
	}

	adapter := llm.NewOpenAI(llmLog, llmClient, "", nil, opts...)

	return checkeruc.NewChecker(adapter), llmCfg.Name, nil
}
