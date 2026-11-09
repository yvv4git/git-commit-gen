package checker

import (
	"context"

	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Checker struct {
	checker ports.ProviderChecker
}

func NewChecker(checker ports.ProviderChecker) *Checker {
	return &Checker{checker: checker}
}

func (c *Checker) Check(ctx context.Context) (*ports.ProviderCheckResult, error) {
	return c.checker.CheckProvider(ctx)
}
