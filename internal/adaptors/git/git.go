package git

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	plumbingobject "github.com/go-git/go-git/v5/plumbing/object"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Git struct {
	commandTimeout time.Duration
}

func New() *Git {
	return &Git{commandTimeout: defaultCommandTimeout}
}

func (g *Git) LoadDiff(ctx context.Context, params *ports.LoadDiffParams) (*ports.LoadDiffResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	_, headCommit, mergeBase, err := g.resolveHistory(repo, params.BaseBranch)
	if err != nil {
		return nil, err
	}

	if headCommit.Hash == mergeBase.Hash {
		return nil, domain.ErrNoCommitsToUpdate
	}

	patch, err := mergeBase.Patch(headCommit)
	if err != nil {
		return nil, fmt.Errorf("compute diff: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	diff := patch.String()
	if params.MaxBytes > 0 && len(diff) > params.MaxBytes {
		return nil, fmt.Errorf("%w: %d > %d bytes", domain.ErrDiffTooLarge, len(diff), params.MaxBytes)
	}

	return &ports.LoadDiffResult{Diff: diff}, nil
}

func (*Git) CurrentBranch(ctx context.Context, _ *ports.CurrentBranchParams) (*ports.CurrentBranchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	headRef, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return nil, fmt.Errorf("get HEAD reference: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &ports.CurrentBranchResult{Value: headRef.Name().Short()}, nil
}

func (g *Git) UpdateFirstCommit(ctx context.Context, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	headRef, headCommit, mergeBase, err := g.resolveHistory(repo, params.BaseBranch)
	if err != nil {
		return nil, err
	}

	commits, err := g.branchCommits(headCommit, mergeBase)
	if err != nil {
		return nil, err
	}

	backupRef := newBackupRef()
	if err := repo.Storer.SetReference(
		plumbing.NewHashReference(plumbing.ReferenceName(backupRef), headCommit.Hash),
	); err != nil {
		return nil, fmt.Errorf("create backup ref: %w", err)
	}

	signingEnabled, err := g.commitSigningEnabled(ctx)
	if err != nil {
		return nil, err
	}

	var result *ports.UpdateFirstCommitResult
	if signingEnabled {
		result, err = g.updateFirstCommitWithGit(ctx, commits, headRef, params)
	} else {
		result, err = g.updateFirstCommitWithGoGit(ctx, repo, commits, headRef, params)
	}
	if err != nil {
		return nil, err
	}

	result.BackupRef = backupRef

	return result, nil
}

func newBackupRef() string {
	return "refs/git-commit-gen/backup/" + time.Now().UTC().Format("20060102T150405.000000000Z")
}

func (g *Git) FirstCommitMessage(ctx context.Context, params *ports.FirstCommitMessageParams) (*ports.FirstCommitMessageResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	_, headCommit, mergeBase, err := g.resolveHistory(repo, params.BaseBranch)
	if err != nil {
		return nil, err
	}

	commits, err := g.branchCommits(headCommit, mergeBase)
	if err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &ports.FirstCommitMessageResult{CommitMessage: commits[0].Message}, nil
}

func (*Git) resolveHistory(repo *git.Repository, baseBranch string) (*plumbing.Reference, *plumbingobject.Commit, *plumbingobject.Commit, error) {
	headRef, err := repo.Head()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get HEAD ref: %w", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get HEAD commit: %w", err)
	}

	baseRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+baseBranch), true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get base branch ref: %w", err)
	}

	baseCommit, err := repo.CommitObject(baseRef.Hash())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get base commit: %w", err)
	}

	bases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("find merge base: %w", err)
	}
	if len(bases) == 0 {
		return nil, nil, nil, errors.New("no merge base found")
	}

	return headRef, headCommit, bases[0], nil
}

func (*Git) branchCommits(headCommit, mergeBase *plumbingobject.Commit) ([]*plumbingobject.Commit, error) {
	var commits []*plumbingobject.Commit

	current := headCommit
	for current.Hash != mergeBase.Hash {
		commits = append(commits, current)
		if len(current.ParentHashes) == 0 {
			return nil, fmt.Errorf("merge base %s is not reachable through first parents", mergeBase.Hash)
		}

		parent, err := current.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("walk parent: %w", err)
		}

		current = parent
	}

	if len(commits) == 0 {
		return nil, domain.ErrNoCommitsToUpdate
	}

	slices.Reverse(commits)

	return commits, nil
}

func (g *Git) updateFirstCommitWithGit(ctx context.Context, commits []*plumbingobject.Commit, headRef *plumbing.Reference, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	var prevHash string
	for i, commit := range commits {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		message := commit.Message
		if i == 0 {
			message = params.CommitMessage
		}

		parents := commit.ParentHashes
		if i > 0 {
			parents = []plumbing.Hash{plumbing.NewHash(prevHash)}
		}

		hash, err := g.commitTree(ctx, commit.TreeHash, message, parents)
		if err != nil {
			return nil, err
		}

		prevHash = hash
	}

	if _, _, err := g.runGit(ctx, "", "update-ref", headRef.Name().String(), prevHash); err != nil {
		return nil, err
	}

	return &ports.UpdateFirstCommitResult{}, nil
}

func (g *Git) commitTree(ctx context.Context, tree plumbing.Hash, message string, parents []plumbing.Hash) (string, error) {
	args := []string{"commit-tree", "-S", tree.String()}
	for _, parent := range parents {
		args = append(args, "-p", parent.String())
	}

	hash, _, err := g.runGit(ctx, message, args...)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func (g *Git) updateFirstCommitWithGoGit(ctx context.Context, repo *git.Repository, commits []*plumbingobject.Commit, headRef *plumbing.Reference, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	var prevHash plumbing.Hash
	for i, commit := range commits {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		message := commit.Message
		if i == 0 {
			message = params.CommitMessage
		}

		parents := commit.ParentHashes
		if i > 0 {
			parents = []plumbing.Hash{prevHash}
		}

		hash, err := g.storeCommit(repo, commit, message, parents)
		if err != nil {
			return nil, err
		}

		if i == len(commits)-1 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			newRef := plumbing.NewHashReference(headRef.Name(), hash)
			if err := repo.Storer.SetReference(newRef); err != nil {
				return nil, fmt.Errorf("update branch ref: %w", err)
			}
		}

		prevHash = hash
	}

	return &ports.UpdateFirstCommitResult{}, nil
}

func (*Git) storeCommit(repo *git.Repository, commit *plumbingobject.Commit, message string, parents []plumbing.Hash) (plumbing.Hash, error) {
	newCommit := &plumbingobject.Commit{
		Author:       commit.Author,
		Committer:    commit.Committer,
		Message:      message,
		TreeHash:     commit.TreeHash,
		ParentHashes: parents,
	}

	obj := repo.Storer.NewEncodedObject()
	if err := newCommit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("encode commit: %w", err)
	}

	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("store commit: %w", err)
	}

	return hash, nil
}

func (g *Git) commitSigningEnabled(ctx context.Context) (bool, error) {
	value, _, err := g.runGit(ctx, "", "config", "--get", "commit.gpgsign")
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false, err
		}

		return false, nil
	}

	return strings.TrimSpace(value) == "true", nil
}
