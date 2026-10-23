package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	plumbingobject "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Git struct{}

func New() *Git {
	return &Git{}
}

func (g *Git) LoadDiff(_ context.Context, params *ports.LoadDiffParams) (*ports.LoadDiffResult, error) {
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

	return &ports.LoadDiffResult{Diff: patch.String()}, nil
}

func (*Git) CurrentBranch(_ context.Context, _ *ports.CurrentBranchParams) (*ports.CurrentBranchResult, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	headRef, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return nil, fmt.Errorf("get HEAD reference: %w", err)
	}

	return &ports.CurrentBranchResult{Value: headRef.Name().Short()}, nil
}

func (g *Git) UpdateFirstCommit(_ context.Context, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
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

	if g.commitSigningEnabled() {
		return g.updateFirstCommitWithGit(commits, headRef, params)
	}

	return g.updateFirstCommitWithGoGit(repo, commits, headRef, params)
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

func (g *Git) updateFirstCommitWithGit(commits []*plumbingobject.Commit, headRef *plumbing.Reference, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	var prevHash string
	for i, commit := range commits {
		message := commit.Message
		if i == 0 {
			message = params.CommitMessage
		}

		parents := commit.ParentHashes
		if i > 0 {
			parents = []plumbing.Hash{plumbing.NewHash(prevHash)}
		}

		hash, err := g.commitTree(commit.TreeHash, message, parents)
		if err != nil {
			return nil, err
		}

		prevHash = hash
	}

	cmd := exec.Command("git", "update-ref", headRef.Name().String(), prevHash)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git update-ref: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return &ports.UpdateFirstCommitResult{}, nil
}

func (*Git) commitTree(tree plumbing.Hash, message string, parents []plumbing.Hash) (string, error) {
	args := []string{"commit-tree", "-S", tree.String()}
	for _, parent := range parents {
		args = append(args, "-p", parent.String())
	}

	cmd := exec.Command("git", args...)
	cmd.Stdin = strings.NewReader(message)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git commit-tree: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Git) updateFirstCommitWithGoGit(repo *git.Repository, commits []*plumbingobject.Commit, headRef *plumbing.Reference, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	var prevHash plumbing.Hash
	for i, commit := range commits {
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

func (*Git) commitSigningEnabled() bool {
	cmd := exec.Command("git", "config", "--get", "commit.gpgsign")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(stdout.String()) == "true"
}
