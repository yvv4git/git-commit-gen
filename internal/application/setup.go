package application

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yvv4git/git-commit-gen/internal/adaptors/fs"
	"github.com/yvv4git/git-commit-gen/internal/core/setup"
)

type SetupOptions struct {
	ConfigContent string
	RulesContent  string
	Force         bool
}

type SetupApplication struct {
	ctx     context.Context
	stop    context.CancelFunc
	service *setup.Setup
	opts    SetupOptions
}

func NewSetupApplication(parent context.Context, opts SetupOptions) *SetupApplication {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)

	fileSystem := fs.NewFS()

	return &SetupApplication{
		ctx:     ctx,
		stop:    stop,
		service: setup.NewSetup(fileSystem, fileSystem),
		opts:    opts,
	}
}

func (a *SetupApplication) Run() (*setup.SetupResult, error) {
	return a.service.Run(a.ctx, &setup.SetupParams{
		ConfigContent: a.opts.ConfigContent,
		RulesContent:  a.opts.RulesContent,
		Force:         a.opts.Force,
	})
}

func (a *SetupApplication) Shutdown() {
	a.stop()
}
