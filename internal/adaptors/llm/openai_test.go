package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
	"go.uber.org/zap"
)

const (
	testBaseBranch    = "main"
	testWorkBranch    = "feature"
	testDiff          = "@@ -1 +1 @@"
	testCommitMessage = "TASK-14: add agent"
)

var errGenerate = errors.New("generate failed")

type fakeGit struct {
	branch     string
	diff       string
	branchErr  error
	diffErr    error
	updateErr  error
	diffParams *ports.LoadDiffParams
	updates    []*ports.UpdateFirstCommitParams
}

func (f *fakeGit) CurrentBranch(_ context.Context, _ *ports.CurrentBranchParams) (*ports.CurrentBranchResult, error) {
	if f.branchErr != nil {
		return nil, f.branchErr
	}

	return &ports.CurrentBranchResult{Value: f.branch}, nil
}

func (f *fakeGit) LoadDiff(_ context.Context, params *ports.LoadDiffParams) (*ports.LoadDiffResult, error) {
	f.diffParams = params
	if f.diffErr != nil {
		return nil, f.diffErr
	}

	return &ports.LoadDiffResult{Diff: f.diff}, nil
}

func (f *fakeGit) UpdateFirstCommit(_ context.Context, params *ports.UpdateFirstCommitParams) (*ports.UpdateFirstCommitResult, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	f.updates = append(f.updates, params)

	return &ports.UpdateFirstCommitResult{}, nil
}

type fakeGenerator struct {
	responses []*llms.ContentResponse
	err       error
	calls     int
	messages  []llms.MessageContent
}

func (f *fakeGenerator) GenerateContent(_ context.Context, messages []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	f.calls++
	f.messages = messages

	if f.err != nil {
		return nil, f.err
	}

	index := f.calls - 1
	if index >= len(f.responses) {
		return nil, errors.New("unexpected generate content call")
	}

	return f.responses[index], nil
}

func newTestAdapter(gitClient ports.Git, generator contentGenerator, rules string) *OpenAI {
	return NewOpenAI(
		zap.NewNop(),
		generator,
		rules,
		gitClient,
		Settings{
			BaseBranch: testBaseBranch,
			Retry: RetrySettings{
				Retries:      0,
				InitInterval: time.Millisecond,
				MaxInterval:  time.Millisecond,
			},
		},
	)
}

func newToolCall(name, arguments string) llms.ToolCall {
	return llms.ToolCall{
		ID:           name,
		Type:         "function",
		FunctionCall: &llms.FunctionCall{Name: name, Arguments: arguments},
	}
}

func newToolResponse(calls ...llms.ToolCall) *llms.ContentResponse {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{ToolCalls: calls}},
	}
}

func newTextResponse(content string) *llms.ContentResponse {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{Content: content}},
	}
}

func newUpdateCall() llms.ToolCall {
	return newToolCall(toolUpdateFirstCommit, `{"commit_message":"`+testCommitMessage+`"}`)
}

func TestGenCommitDescriptionFullCycle(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch, diff: testDiff}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolGetCurrentBranch, "{}")),
		newToolResponse(newToolCall(toolLoadDiff, "{}")),
		newToolResponse(newUpdateCall()),
		newTextResponse("Done"),
	}}

	result, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, testCommitMessage, result.Value)
	assert.Equal(t, testBaseBranch, gitClient.diffParams.BaseBranch)
	require.Len(t, gitClient.updates, 1)
	assert.Equal(t, testCommitMessage, gitClient.updates[0].CommitMessage)
	assert.Equal(t, testBaseBranch, gitClient.updates[0].BaseBranch)
}

func TestNewAgentRun(t *testing.T) {
	const rules = "Use the TASK prefix."

	run := newTestAdapter(&fakeGit{}, &fakeGenerator{}, rules).newAgentRun()

	require.Len(t, run.messages, 2)
	assert.Equal(t, llms.ChatMessageTypeSystem, run.messages[0].Role)
	assert.Equal(t, []llms.ContentPart{
		llms.TextContent{Text: systemPrompt},
		llms.TextContent{Text: "Commit message rules:\n" + rules},
	}, run.messages[0].Parts)
	assert.Equal(t, llms.TextParts(llms.ChatMessageTypeHuman, taskPrompt), run.messages[1])
}

func TestGenCommitDescriptionToolError(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch, diffErr: domain.ErrNoCommitsToUpdate}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolLoadDiff, "{}")),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionWithoutUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newTextResponse("Nothing to update"),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrCommitNotUpdated)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionRepeatedUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
		newToolResponse(newUpdateCall()),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrCommitAlreadyUpdated)
	require.Len(t, gitClient.updates, 1)
}

func TestGenCommitDescriptionOnBaseBranch(t *testing.T) {
	gitClient := &fakeGit{branch: testBaseBranch}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrOnBaseBranch)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionEmptyMessage(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolUpdateFirstCommit, `{}`)),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrEmptyCommitMessage)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionMalformedArguments(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolUpdateFirstCommit, "{")),
	}}

	_, err := newTestAdapter(gitClient, generator, "").GenCommitDescription(context.Background())

	require.Error(t, err)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionUnknownTool(t *testing.T) {
	generator := &fakeGenerator{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall("DeleteEverything", "{}")),
	}}

	_, err := newTestAdapter(&fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrUnknownTool)
}

func TestGenCommitDescriptionNoChoices(t *testing.T) {
	generator := &fakeGenerator{responses: []*llms.ContentResponse{{}}}

	_, err := newTestAdapter(&fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrNoResponseChoices)
}

func TestGenCommitDescriptionGenerateError(t *testing.T) {
	generator := &fakeGenerator{err: errGenerate}

	_, err := newTestAdapter(&fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, errGenerate)
}
