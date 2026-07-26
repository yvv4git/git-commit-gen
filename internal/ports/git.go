package ports

import "context"

type (
	LoadDiffParams struct {
		BaseBranch string
	}
	LoadDiffResult struct {
		Diff string
	}
)

type (
	CurrentBranchParams struct {
		BranchName string
	}
	CurrentBranchResult struct {
		Value string
	}
)

type (
	UpdateFirstCommitParams struct {
		CommitMessage string
		BaseBranch    string
	}
	UpdateFirstCommitResult struct{}
)

type Git interface {
	LoadDiff(ctx context.Context, params *LoadDiffParams) (*LoadDiffResult, error)
	CurrentBranch(ctx context.Context, params *CurrentBranchParams) (*CurrentBranchResult, error)
	UpdateFirstCommit(ctx context.Context, params *UpdateFirstCommitParams) (*UpdateFirstCommitResult, error)
}
