package generator

import (
	"context"
	"fmt"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Generator struct {
	baseBranch string
	git        ports.Git
	llm        ports.LLM
}

func NewGenerator(baseBranch string, git ports.Git, llm ports.LLM) *Generator {
	return &Generator{
		baseBranch: baseBranch,
		git:        git,
		llm:        llm,
	}
}

func (g *Generator) Gen(ctx context.Context) (*ports.GenCommitDescriptionResult, error) {
	current, err := g.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}
	if current.Value == g.baseBranch {
		return nil, fmt.Errorf("%w: %q", domain.ErrOnBaseBranch, current.Value)
	}

	result, err := g.llm.GenCommitDescription(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}
