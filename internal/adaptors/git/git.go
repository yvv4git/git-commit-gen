package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	plumbingobject "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

type Git struct {
}

func New() *Git {
	return &Git{}
}

func (g *Git) LoadDiff(ctx context.Context, params *ports.LoadDiffParams) (*ports.LoadDiffResult, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD ref: %w", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("get HEAD commit: %w", err)
	}

	baseRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+params.BaseBranch), true)
	if err != nil {
		return nil, fmt.Errorf("get base branch ref: %w", err)
	}

	baseCommit, err := repo.CommitObject(baseRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("get base commit: %w", err)
	}

	bases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return nil, fmt.Errorf("find merge base: %w", err)
	}
	if len(bases) == 0 {
		return nil, fmt.Errorf("no merge base found")
	}

	patch, err := bases[0].Patch(headCommit)
	if err != nil {
		return nil, fmt.Errorf("compute diff: %w", err)
	}

	return &ports.LoadDiffResult{Diff: patch.String()}, nil
}

func (g *Git) CurrentBranch(ctx context.Context, params *ports.CurrentBranchParams) (*ports.CurrentBranchResult, error) {
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

func (g *Git) UpdateFirstCommit(ctx context.Context, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD ref: %w", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("get HEAD commit: %w", err)
	}

	baseRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+params.BaseBranch), true)
	if err != nil {
		return nil, fmt.Errorf("get base branch ref: %w", err)
	}

	baseCommit, err := repo.CommitObject(baseRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("get base commit: %w", err)
	}

	bases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return nil, fmt.Errorf("find merge base: %w", err)
	}
	if len(bases) == 0 {
		return nil, fmt.Errorf("no merge base found")
	}
	mergeBase := bases[0]

	var commits []*plumbingobject.Commit
	current := headCommit
	for {
		commits = append(commits, current)
		if len(current.ParentHashes) == 0 {
			break
		}
		if current.ParentHashes[0] == mergeBase.Hash {
			break
		}
		parent, err := current.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("walk parent: %w", err)
		}
		current = parent
	}

	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	if len(commits) == 0 {
		return &ports.UpdateFirstCommitResult{}, nil
	}

	if commitSigningEnabled() {
		return g.updateFirstCommitWithGit(commits, headRef, params)
	}

	return g.updateFirstCommitWithGoGit(repo, commits, headRef, params)
}

func (g *Git) updateFirstCommitWithGoGit(
	repo *git.Repository,
	commits []*plumbingobject.Commit,
	headRef *plumbing.Reference,
	params *ports.UpdateFirstCommitParams,
) (*ports.UpdateFirstCommitResult, error) {
	var prevHash plumbing.Hash
	for i, c := range commits {
		msg := c.Message
		if i == 0 {
			msg = params.CommitMessage
		}

		parents := c.ParentHashes
		if i > 0 {
			parents = []plumbing.Hash{prevHash}
		}

		newCommit := &plumbingobject.Commit{
			Author:       c.Author,
			Committer:    c.Committer,
			Message:      msg,
			TreeHash:     c.TreeHash,
			ParentHashes: parents,
		}

		obj := repo.Storer.NewEncodedObject()
		if err := newCommit.Encode(obj); err != nil {
			return nil, fmt.Errorf("encode commit: %w", err)
		}

		hash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			return nil, fmt.Errorf("store commit: %w", err)
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

func (g *Git) updateFirstCommitWithGit(
	commits []*plumbingobject.Commit,
	headRef *plumbing.Reference,
	params *ports.UpdateFirstCommitParams,
) (*ports.UpdateFirstCommitResult, error) {
	var prevHash string
	for i, c := range commits {
		msg := c.Message
		if i == 0 {
			msg = params.CommitMessage
		}

		args := []string{"commit-tree", "-S", c.TreeHash.String()}
		if i == 0 {
			for _, ph := range c.ParentHashes {
				args = append(args, "-p", ph.String())
			}
		} else {
			args = append(args, "-p", prevHash)
		}

		cmd := exec.Command("git", args...)
		cmd.Stdin = strings.NewReader(msg)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("git commit-tree: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
		}

		prevHash = strings.TrimSpace(stdout.String())
	}

	cmd := exec.Command("git", "update-ref", headRef.Name().String(), prevHash)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git update-ref: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return &ports.UpdateFirstCommitResult{}, nil
}

// commitSigningEnabled returns true if commit.gpgsign is true in git config.
func commitSigningEnabled() bool {
	cmd := exec.Command("git", "config", "--get", "commit.gpgsign")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(stdout.String()) == "true"
}
