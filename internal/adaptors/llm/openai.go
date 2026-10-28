package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
	"github.com/yvv4git/git-commit-gen/pkg/retry"
	"go.uber.org/zap"
)

const (
	maxAgentTurns        = 12
	maxUpdateReminders   = 2
	updateCommitReminder = "You have not updated the first commit yet. Do not reply with the commit message. Call UpdateFirstCommit now with the generated commit_message."
)

type RetrySettings struct {
	Retries      uint64
	InitInterval time.Duration
	MaxInterval  time.Duration
}

type Settings struct {
	BaseBranch string
	Retry      RetrySettings
}

type OpenAI struct {
	log        *zap.Logger
	client     *openai.LLM
	rules      string
	git        ports.Git
	baseBranch string
	retry      RetrySettings
}

type agentRun struct {
	messages []llms.MessageContent
	updated  bool
	verified bool
	message  string
}

type contentCall struct {
	ctx      context.Context
	log      *zap.Logger
	client   *openai.LLM
	messages []llms.MessageContent
	tools    []llms.Tool
	retries  uint64
	attempt  int
}

func NewOpenAI(
	log *zap.Logger,
	client *openai.LLM,
	rules string,
	git ports.Git,
	settings Settings,
) *OpenAI {
	return &OpenAI{
		log:        log,
		client:     client,
		rules:      rules,
		git:        git,
		baseBranch: settings.BaseBranch,
		retry:      settings.Retry,
	}
}

func (l *OpenAI) GenCommitDescription(ctx context.Context) (*ports.GenCommitDescriptionResult, error) {
	run := l.newAgentRun()
	tools := Tools()
	reminders := 0

	for turn := 0; turn < maxAgentTurns; turn++ {
		resp, err := l.generateContent(ctx, run.messages, tools)
		if err != nil {
			return nil, err
		}

		choice, err := run.firstChoice(resp)
		if err != nil {
			return nil, err
		}

		if len(choice.ToolCalls) == 0 {
			if !run.updated && reminders < maxUpdateReminders {
				reminders++

				run.appendAssistant(choice)
				run.appendReminder(updateCommitReminder)

				continue
			}

			return l.result(ctx, run)
		}

		if err := l.executeToolCalls(ctx, run, choice); err != nil {
			return nil, err
		}
	}

	return l.result(ctx, run)
}

func (l *OpenAI) newAgentRun() *agentRun {
	systemParts := []string{systemPrompt}
	if l.rules != "" {
		systemParts = append(systemParts, "Commit message rules:\n"+l.rules)
	}

	return &agentRun{
		messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemParts...),
			llms.TextParts(llms.ChatMessageTypeHuman, taskPrompt),
		},
	}
}

func (l *OpenAI) executeToolCalls(ctx context.Context, run *agentRun, choice *llms.ContentChoice) error {
	run.appendAssistant(choice)

	for _, call := range choice.ToolCalls {
		result, err := l.executeToolCall(ctx, run, call)
		if err != nil {
			return err
		}

		run.appendToolResponse(call, result)
	}

	return nil
}

func (l *OpenAI) executeToolCall(ctx context.Context, run *agentRun, call llms.ToolCall) (string, error) {
	if call.FunctionCall == nil {
		return "", fmt.Errorf("%w: %s", domain.ErrToolCallWithoutFunction, call.ID)
	}

	l.log.Info("Calling tool", zap.String("tool", call.FunctionCall.Name))

	switch call.FunctionCall.Name {
	case toolGetCurrentBranch:
		return l.getCurrentBranch(ctx)
	case toolLoadDiff:
		return l.loadDiff(ctx)
	case toolUpdateFirstCommit:
		return l.updateFirstCommit(ctx, run, call.FunctionCall.Arguments)
	case toolVerifyFirstCommit:
		return l.verifyFirstCommit(ctx, run)
	default:
		return "", fmt.Errorf("%w: %s", domain.ErrUnknownTool, call.FunctionCall.Name)
	}
}

func (l *OpenAI) getCurrentBranch(ctx context.Context) (string, error) {
	res, err := l.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return res.Value, nil
}

func (l *OpenAI) loadDiff(ctx context.Context) (string, error) {
	res, err := l.git.LoadDiff(ctx, &ports.LoadDiffParams{BaseBranch: l.baseBranch})
	if err != nil {
		return "", fmt.Errorf("load diff: %w", err)
	}

	return res.Diff, nil
}

func (l *OpenAI) updateFirstCommit(ctx context.Context, run *agentRun, arguments string) (string, error) {
	if run.updated {
		return "", domain.ErrCommitAlreadyUpdated
	}

	var input updateFirstCommitArgs
	if err := input.parse(arguments); err != nil {
		return "", err
	}

	if err := l.ensureNotBaseBranch(ctx); err != nil {
		return "", err
	}

	if err := l.applyFirstCommit(ctx, input.CommitMessage); err != nil {
		return "", err
	}

	run.updated = true
	run.message = input.CommitMessage

	return "First commit updated", nil
}

func (l *OpenAI) ensureNotBaseBranch(ctx context.Context) error {
	current, err := l.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if current.Value == l.baseBranch {
		return fmt.Errorf("%w: %q", domain.ErrOnBaseBranch, current.Value)
	}

	return nil
}

func (l *OpenAI) applyFirstCommit(ctx context.Context, message string) error {
	_, err := l.git.UpdateFirstCommit(ctx, &ports.UpdateFirstCommitParams{
		CommitMessage: message,
		BaseBranch:    l.baseBranch,
	})
	if err != nil {
		return fmt.Errorf("update first commit: %w", err)
	}

	return nil
}

func (l *OpenAI) verifyFirstCommit(ctx context.Context, run *agentRun) (string, error) {
	if err := l.checkFirstCommit(ctx, run); err != nil {
		return "", err
	}

	return "First commit verified", nil
}

func (l *OpenAI) checkFirstCommit(ctx context.Context, run *agentRun) error {
	if !run.updated {
		return domain.ErrCommitNotUpdated
	}

	res, err := l.git.FirstCommitMessage(ctx, &ports.FirstCommitMessageParams{BaseBranch: l.baseBranch})
	if err != nil {
		return fmt.Errorf("load first commit message: %w", err)
	}

	got := strings.TrimSpace(res.CommitMessage)
	want := strings.TrimSpace(run.message)
	if got != want {
		return fmt.Errorf("%w: got %q, want %q", domain.ErrFirstCommitMismatch, got, want)
	}

	run.verified = true

	return nil
}

func (l *OpenAI) generateContent(ctx context.Context, messages []llms.MessageContent, tools []llms.Tool) (*llms.ContentResponse, error) {
	call := &contentCall{
		ctx:      ctx,
		log:      l.log,
		client:   l.client,
		messages: messages,
		tools:    tools,
		retries:  l.retry.Retries,
	}

	resp, err := retry.RetryWithResult(ctx, call.generate,
		retry.WithRetries(l.retry.Retries),
		retry.WithInitInterval(l.retry.InitInterval),
		retry.WithMaxInterval(l.retry.MaxInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	return resp, nil
}

func (c *contentCall) generate() (*llms.ContentResponse, error) {
	c.attempt++

	resp, err := c.client.GenerateContent(c.ctx, c.messages, llms.WithTools(c.tools))
	if err != nil {
		c.log.Warn("Generate content",
			zap.Int("attempt", c.attempt),
			zap.Uint64("max_attempts", c.retries+1),
			zap.Error(err))
	}

	return resp, err
}

func (r *agentRun) firstChoice(resp *llms.ContentResponse) (*llms.ContentChoice, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return nil, domain.ErrNoResponseChoices
	}

	return resp.Choices[0], nil
}

func (r *agentRun) appendAssistant(choice *llms.ContentChoice) {
	parts := make([]llms.ContentPart, 0, len(choice.ToolCalls)+1)
	if choice.Content != "" {
		parts = append(parts, llms.TextPart(choice.Content))
	}

	for _, call := range choice.ToolCalls {
		parts = append(parts, call)
	}

	r.messages = append(r.messages, llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: parts,
	})
}

func (r *agentRun) appendToolResponse(call llms.ToolCall, content string) {
	r.messages = append(r.messages, llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: call.ID,
				Name:       call.FunctionCall.Name,
				Content:    content,
			},
		},
	})
}

func (r *agentRun) appendReminder(content string) {
	r.messages = append(r.messages, llms.TextParts(llms.ChatMessageTypeHuman, content))
}

func (l *OpenAI) result(ctx context.Context, run *agentRun) (*ports.GenCommitDescriptionResult, error) {
	if !run.updated {
		return nil, domain.ErrCommitNotUpdated
	}

	if !run.verified {
		if err := l.checkFirstCommit(ctx, run); err != nil {
			return nil, err
		}
	}

	return &ports.GenCommitDescriptionResult{Value: run.message}, nil
}

func (a *updateFirstCommitArgs) parse(arguments string) error {
	if err := json.Unmarshal([]byte(arguments), a); err != nil {
		return fmt.Errorf("parse update first commit args: %w", err)
	}
	if a.CommitMessage == "" {
		return domain.ErrEmptyCommitMessage
	}

	return nil
}
