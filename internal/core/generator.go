package core

import (
	"context"
	"fmt"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Generator struct {
	baseBranch string
	visual     bool
	git        ports.Git
	llm        ports.LLM
}

func NewGenerator(baseBranch string, visual bool, git ports.Git, llm ports.LLM) *Generator {
	return &Generator{
		baseBranch: baseBranch,
		visual:     visual,
		git:        git,
		llm:        llm,
	}
}

func (g *Generator) Gen(ctx context.Context) error {
	current, err := g.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if current.Value == g.baseBranch {
		return fmt.Errorf("%w: %q", domain.ErrOnBaseBranch, current.Value)
	}

	result, err := g.llm.GenCommitDescription(ctx)
	if err != nil {
		return err
	}

	if g.visual {
		fmt.Println(result.Value)
	}

	return nil
}
