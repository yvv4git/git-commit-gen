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

type (
	FirstCommitMessageParams struct {
		BaseBranch string
	}
	FirstCommitMessageResult struct {
		CommitMessage string
	}
)

type Git interface {
	LoadDiff(ctx context.Context, params *LoadDiffParams) (*LoadDiffResult, error)
	CurrentBranch(ctx context.Context, params *CurrentBranchParams) (*CurrentBranchResult, error)
	UpdateFirstCommit(ctx context.Context, params *UpdateFirstCommitParams) (*UpdateFirstCommitResult, error)
	FirstCommitMessage(ctx context.Context, params *FirstCommitMessageParams) (*FirstCommitMessageResult, error)
}
