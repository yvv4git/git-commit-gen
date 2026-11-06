package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/yvv4git/git-commit-gen/pkg/async"
)

const defaultCommandTimeout = 30 * time.Second

type command struct {
	stdin string
	args  []string
}

type commandResult struct {
	stdout string
	stderr string
}

func newWithTimeout(timeout time.Duration) *Git {
	return &Git{commandTimeout: timeout}
}

func (g *Git) runGit(ctx context.Context, stdin string, args ...string) (string, string, error) {
	cmd := command{stdin: stdin, args: args}

	result, err := async.RunWithTimeoutAndResult(ctx, cmd.run, g.timeout())
	if err != nil {
		return "", result.stderr, cmd.wrapError(err)
	}

	return result.stdout, result.stderr, nil
}

func (c command) run(ctx context.Context) (commandResult, error) {
	cmd := exec.CommandContext(ctx, "git", c.args...)
	configureCommand(cmd)
	if c.stdin != "" {
		cmd.Stdin = strings.NewReader(c.stdin)
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := commandResult{
		stdout: strings.TrimSpace(stdout.String()),
		stderr: strings.TrimSpace(stderr.String()),
	}
	if err == nil {
		return result, nil
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return result, ctxErr
	}

	return result, fmt.Errorf("git %s: %w\nstderr: %s", c.name(), err, result.stderr)
}

func (c command) wrapError(err error) error {
	if errors.Is(err, async.ErrTimeout) {
		return fmt.Errorf("git %s: %w", c.name(), context.DeadlineExceeded)
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("git %s: %w", c.name(), err)
	}

	return err
}

func (c command) name() string {
	if len(c.args) == 0 {
		return "git"
	}

	return c.args[0]
}

func (g *Git) timeout() time.Duration {
	if g.commandTimeout <= 0 {
		return defaultCommandTimeout
	}

	return g.commandTimeout
}
