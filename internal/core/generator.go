package core

import (
	"context"
	"fmt"

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
	cur, err := g.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if cur.Value == g.baseBranch {
		return fmt.Errorf("cannot generate commit on base branch %q", g.baseBranch)
	}

	gitRes, err := g.git.LoadDiff(ctx, &ports.LoadDiffParams{
		BaseBranch: g.baseBranch,
	})
	if err != nil {
		return err
	}

	genRes, err := g.llm.GenCommitDescription(ctx, &ports.GenCommitDescriptionParams{
		Diff: gitRes.Diff,
	})
	if err != nil {
		return err
	}

	msg := genRes.Value

	if g.visual {
		fmt.Println(msg)
	}

	_, err = g.git.UpdateFirstCommit(ctx, &ports.UpdateFirstCommitParams{
		CommitMessage: msg,
		BaseBranch:    g.baseBranch,
	})
	if err != nil {
		return err
	}

	return nil
}
