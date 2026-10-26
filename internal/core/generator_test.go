package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

var (
	errCurrentBranch = errors.New("current branch failed")
	errAgent         = errors.New("agent failed")
)

type fakeGit struct {
	branch     string
	currentErr error
}

func (f *fakeGit) CurrentBranch(_ context.Context, _ *ports.CurrentBranchParams) (*ports.CurrentBranchResult, error) {
	if f.currentErr != nil {
		return nil, f.currentErr
	}

	return &ports.CurrentBranchResult{Value: f.branch}, nil
}

func (*fakeGit) LoadDiff(_ context.Context, _ *ports.LoadDiffParams) (*ports.LoadDiffResult, error) {
	return &ports.LoadDiffResult{}, nil
}

func (*fakeGit) UpdateFirstCommit(_ context.Context, _ *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	return &ports.UpdateFirstCommitResult{}, nil
}

func (*fakeGit) FirstCommitMessage(_ context.Context, _ *ports.FirstCommitMessageParams) (*ports.FirstCommitMessageResult, error) {
	return &ports.FirstCommitMessageResult{}, nil
}

type fakeLLM struct {
	result *ports.GenCommitDescriptionResult
	err    error
	calls  int
}

func (f *fakeLLM) GenCommitDescription(_ context.Context) (*ports.GenCommitDescriptionResult, error) {
	f.calls++

	return f.result, f.err
}

func TestGenOnBaseBranch(t *testing.T) {
	llm := &fakeLLM{}
	generator := NewGenerator("main", false, &fakeGit{branch: "main"}, llm)

	err := generator.Gen(context.Background())

	require.ErrorIs(t, err, domain.ErrOnBaseBranch)
	assert.Zero(t, llm.calls)
}

func TestGenCurrentBranchError(t *testing.T) {
	llm := &fakeLLM{}
	generator := NewGenerator("main", false, &fakeGit{currentErr: errCurrentBranch}, llm)

	err := generator.Gen(context.Background())

	require.ErrorIs(t, err, errCurrentBranch)
	assert.Zero(t, llm.calls)
}

func TestGenAgentError(t *testing.T) {
	llm := &fakeLLM{err: errAgent}
	generator := NewGenerator("main", false, &fakeGit{branch: "feature"}, llm)

	err := generator.Gen(context.Background())

	require.ErrorIs(t, err, errAgent)
	assert.Equal(t, 1, llm.calls)
}

func TestGenSuccess(t *testing.T) {
	llm := &fakeLLM{result: &ports.GenCommitDescriptionResult{Value: "TASK-14: add agent"}}
	generator := NewGenerator("main", false, &fakeGit{branch: "feature"}, llm)

	err := generator.Gen(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, llm.calls)
}
