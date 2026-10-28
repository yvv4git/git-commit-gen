package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
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
	branch             string
	diff               string
	firstCommitMessage string
	branchErr          error
	diffErr            error
	updateErr          error
	firstCommitErr     error
	keepFirstCommit    bool
	diffParams         *ports.LoadDiffParams
	updates            []*ports.UpdateFirstCommitParams
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
	if !f.keepFirstCommit {
		f.firstCommitMessage = params.CommitMessage
	}

	return &ports.UpdateFirstCommitResult{}, nil
}

func (f *fakeGit) FirstCommitMessage(_ context.Context, _ *ports.FirstCommitMessageParams) (*ports.FirstCommitMessageResult, error) {
	if f.firstCommitErr != nil {
		return nil, f.firstCommitErr
	}

	return &ports.FirstCommitMessageResult{CommitMessage: f.firstCommitMessage}, nil
}

type fakeTransport struct {
	responses []*llms.ContentResponse
	err       error
	calls     int
}

type apiResponse struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Model   string      `json:"model"`
	Choices []apiChoice `json:"choices"`
}

type apiChoice struct {
	Index   int        `json:"index"`
	Message apiMessage `json:"message"`
}

type apiMessage struct {
	Role      string        `json:"role"`
	Content   string        `json:"content,omitempty"`
	ToolCalls []apiToolCall `json:"tool_calls,omitempty"`
}

type apiToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function apiFunction `json:"function"`
}

type apiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (f *fakeTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}

	index := f.calls - 1
	if index >= len(f.responses) {
		return nil, errors.New("unexpected generate content call")
	}

	body, err := f.responseBody(index)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func (f *fakeTransport) responseBody(index int) ([]byte, error) {
	source := f.responses[index]
	response := apiResponse{ID: "test", Object: "chat.completion", Model: "test"}

	for _, choice := range source.Choices {
		message := apiMessage{Role: "assistant", Content: choice.Content}
		for _, call := range choice.ToolCalls {
			message.ToolCalls = append(message.ToolCalls, apiToolCall{
				ID:       call.ID,
				Type:     "function",
				Function: apiFunction{Name: call.FunctionCall.Name, Arguments: call.FunctionCall.Arguments},
			})
		}

		response.Choices = append(response.Choices, apiChoice{Message: message})
	}

	return json.Marshal(response)
}

func newTestAdapter(t *testing.T, gitClient ports.Git, transport http.RoundTripper, rules string) *OpenAI {
	t.Helper()

	client, err := openai.New(
		openai.WithBaseURL("http://test"),
		openai.WithToken("token"),
		openai.WithModel("test"),
		openai.WithHTTPClient(&http.Client{Transport: transport}),
	)
	require.NoError(t, err)

	return NewOpenAI(
		zap.NewNop(),
		client,
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

func newVerifyCall() llms.ToolCall {
	return newToolCall(toolVerifyFirstCommit, "{}")
}

func TestGenCommitDescriptionFullCycle(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch, diff: testDiff}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolGetCurrentBranch, "{}")),
		newToolResponse(newToolCall(toolLoadDiff, "{}")),
		newToolResponse(newUpdateCall()),
		newToolResponse(newVerifyCall()),
		newTextResponse("Done"),
	}}

	result, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

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

	run := newTestAdapter(t, &fakeGit{}, &fakeTransport{}, rules).newAgentRun()

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
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolLoadDiff, "{}")),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrNoCommitsToUpdate)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionWithoutUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newTextResponse("Nothing to update"),
		newTextResponse("Nothing to update"),
		newTextResponse("Nothing to update"),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrCommitNotUpdated)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionRemindsToUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newTextResponse(testCommitMessage),
		newToolResponse(newUpdateCall()),
		newTextResponse("Done"),
	}}

	result, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.NoError(t, err)
	assert.Equal(t, testCommitMessage, result.Value)
	require.Len(t, gitClient.updates, 1)
}

func TestGenCommitDescriptionAutoVerify(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
		newTextResponse("Done"),
	}}

	result, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.NoError(t, err)
	assert.Equal(t, testCommitMessage, result.Value)
	require.Len(t, gitClient.updates, 1)
}

func TestGenCommitDescriptionVerifyBeforeUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newVerifyCall()),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrCommitNotUpdated)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionVerifyMismatch(t *testing.T) {
	gitClient := &fakeGit{
		branch:             testWorkBranch,
		firstCommitMessage: "Old message",
		keepFirstCommit:    true,
	}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
		newToolResponse(newVerifyCall()),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrFirstCommitMismatch)
	require.Len(t, gitClient.updates, 1)
}

func TestGenCommitDescriptionRepeatedUpdate(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
		newToolResponse(newUpdateCall()),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrCommitAlreadyUpdated)
	require.Len(t, gitClient.updates, 1)
}

func TestGenCommitDescriptionOnBaseBranch(t *testing.T) {
	gitClient := &fakeGit{branch: testBaseBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newUpdateCall()),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrOnBaseBranch)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionEmptyMessage(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolUpdateFirstCommit, `{}`)),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrEmptyCommitMessage)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionMalformedArguments(t *testing.T) {
	gitClient := &fakeGit{branch: testWorkBranch}
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall(toolUpdateFirstCommit, "{")),
	}}

	_, err := newTestAdapter(t, gitClient, generator, "").GenCommitDescription(context.Background())

	require.Error(t, err)
	assert.Empty(t, gitClient.updates)
}

func TestGenCommitDescriptionUnknownTool(t *testing.T) {
	generator := &fakeTransport{responses: []*llms.ContentResponse{
		newToolResponse(newToolCall("DeleteEverything", "{}")),
	}}

	_, err := newTestAdapter(t, &fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.ErrorIs(t, err, domain.ErrUnknownTool)
}

func TestGenCommitDescriptionNoChoices(t *testing.T) {
	generator := &fakeTransport{responses: []*llms.ContentResponse{{}}}

	_, err := newTestAdapter(t, &fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.EqualError(t, err, "generate content: empty response")
}

func TestGenCommitDescriptionGenerateError(t *testing.T) {
	generator := &fakeTransport{err: errGenerate}

	_, err := newTestAdapter(t, &fakeGit{}, generator, "").GenCommitDescription(context.Background())

	require.EqualError(t, err, "generate content: network error: failed to reach API server")
}
