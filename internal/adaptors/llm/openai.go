package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
	"github.com/yvv4git/git-commit-gen/pkg/retry"
)

const (
	defaultMaxDiffBytes        = 256 * 1024
	defaultMaxRulesBytes       = 32 * 1024
	defaultMaxContextBytes     = 512 * 1024
	defaultMaxAgentTurns       = 12
	defaultMaxToolCallsPerTurn = 8
	defaultRetries             = 3
	defaultRetryInitInterval   = 500 * time.Millisecond
	defaultRetryMaxInterval    = 10 * time.Second
	maxUpdateReminders         = 2
	maxCommitMessageBytes      = 4096
	untrustedDataBegin         = "<UNTRUSTED_DATA>"
	untrustedDataEnd           = "</UNTRUSTED_DATA>"
	diffAlreadyLoaded          = "Diff already loaded. Use the previous LoadDiff result."
	updateCommitReminder       = "You have not updated the first commit yet. Do not reply with the commit message. Call UpdateFirstCommit now with the generated commit_message."
	escapedUntrustedTag        = "[escaped untrusted marker]"
)

var untrustedTagPattern = regexp.MustCompile(`(?i)</?UNTRUSTED_DATA[^>]*>`)

type RetrySettings struct {
	Retries      uint64
	InitInterval time.Duration
	MaxInterval  time.Duration
}

type RetryStatusProvider interface {
	Take() (statusCode int, retryAfter time.Duration)
}

type OpenAIOptions struct {
	baseBranch          string
	retry               RetrySettings
	maxDiffBytes        int
	maxRulesBytes       int
	maxContextBytes     int
	maxAgentTurns       int
	maxToolCallsPerTurn int
	retryStatus         RetryStatusProvider
}

type Option func(*OpenAI)

func WithBaseBranch(value string) Option {
	return func(l *OpenAI) {
		l.options.baseBranch = value
	}
}

func WithRetry(value RetrySettings) Option {
	return func(l *OpenAI) {
		if value.Retries == 0 && value.InitInterval == 0 && value.MaxInterval == 0 {
			return
		}

		l.options.retry = value
	}
}

func WithMaxDiffBytes(value int) Option {
	return func(l *OpenAI) {
		l.options.maxDiffBytes = value
	}
}

func WithMaxRulesBytes(value int) Option {
	return func(l *OpenAI) {
		l.options.maxRulesBytes = value
	}
}

func WithMaxContextBytes(value int) Option {
	return func(l *OpenAI) {
		l.options.maxContextBytes = value
	}
}

func WithMaxAgentTurns(value int) Option {
	return func(l *OpenAI) {
		l.options.maxAgentTurns = value
	}
}

func WithMaxToolCallsPerTurn(value int) Option {
	return func(l *OpenAI) {
		l.options.maxToolCallsPerTurn = value
	}
}

func WithRetryStatus(value RetryStatusProvider) Option {
	return func(l *OpenAI) {
		l.options.retryStatus = value
	}
}

type OpenAI struct {
	log     *zap.Logger
	client  *openai.LLM
	rules   string
	git     ports.Git
	options OpenAIOptions
}

type agentRun struct {
	messages         []llms.MessageContent
	updated          bool
	verified         bool
	diffLoaded       bool
	message          string
	backupRef        string
	promptTokens     int
	completionTokens int
	totalTokens      int
	requests         int
}

type contentCall struct {
	ctx         context.Context
	log         *zap.Logger
	client      *openai.LLM
	messages    []llms.MessageContent
	tools       []llms.Tool
	retries     uint64
	attempt     int
	retryStatus RetryStatusProvider
}

func NewOpenAI(
	log *zap.Logger,
	client *openai.LLM,
	rules string,
	git ports.Git,
	opts ...Option,
) *OpenAI {
	entity := &OpenAI{
		log:    log,
		client: client,
		rules:  rules,
		git:    git,
		options: OpenAIOptions{
			retry: RetrySettings{
				Retries:      defaultRetries,
				InitInterval: defaultRetryInitInterval,
				MaxInterval:  defaultRetryMaxInterval,
			},
			maxDiffBytes:        defaultMaxDiffBytes,
			maxRulesBytes:       defaultMaxRulesBytes,
			maxContextBytes:     defaultMaxContextBytes,
			maxAgentTurns:       defaultMaxAgentTurns,
			maxToolCallsPerTurn: defaultMaxToolCallsPerTurn,
		},
	}

	for _, opt := range opts {
		opt(entity)
	}

	return entity
}

func (l *OpenAI) GenCommitDescription(ctx context.Context) (*ports.GenCommitDescriptionResult, error) {
	if err := l.checkRulesSize(); err != nil {
		return nil, err
	}

	run := l.newAgentRun()
	tools := Tools()
	reminders := 0

	for turn := 0; turn < l.options.maxAgentTurns; turn++ {
		if err := l.checkContextBudget(run.messages); err != nil {
			return nil, err
		}

		l.log.Info("Waiting for LLM response",
			zap.Int("turn", turn+1),
			zap.Int("max_turns", l.options.maxAgentTurns),
			zap.Int("messages", len(run.messages)),
			zap.Int("context_bytes", messageBytes(run.messages)))

		resp, attempts, err := l.generateContent(ctx, run.messages, tools)
		if err != nil {
			return nil, err
		}
		run.requests += attempts

		choice, err := run.firstChoice(resp)
		if err != nil {
			return nil, err
		}
		run.addUsage(choice)

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

	return nil, fmt.Errorf("%w: max %d", domain.ErrAgentTurnsExceeded, l.options.maxAgentTurns)
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

func (l *OpenAI) checkRulesSize() error {
	if len(l.rules) > l.options.maxRulesBytes {
		return fmt.Errorf("%w: %d > %d bytes", domain.ErrRulesTooLarge, len(l.rules), l.options.maxRulesBytes)
	}

	return nil
}

func (l *OpenAI) checkContextBudget(messages []llms.MessageContent) error {
	size := messageBytes(messages)
	if size > l.options.maxContextBytes {
		return fmt.Errorf("%w: %d > %d bytes", domain.ErrContextBudgetExceeded, size, l.options.maxContextBytes)
	}

	return nil
}

func messageBytes(messages []llms.MessageContent) int {
	var size int

	for _, message := range messages {
		for _, part := range message.Parts {
			switch value := part.(type) {
			case llms.TextContent:
				size += len(value.Text)
			case llms.ToolCall:
				if value.FunctionCall != nil {
					size += len(value.FunctionCall.Name) + len(value.FunctionCall.Arguments)
				}
			case llms.ToolCallResponse:
				size += len(value.Name) + len(value.Content)
			}
		}
	}

	return size
}

func (l *OpenAI) executeToolCalls(ctx context.Context, run *agentRun, choice *llms.ContentChoice) error {
	if len(choice.ToolCalls) > l.options.maxToolCallsPerTurn {
		return fmt.Errorf("%w: got %d, max %d",
			domain.ErrTooManyToolCalls, len(choice.ToolCalls), l.options.maxToolCallsPerTurn)
	}

	run.appendAssistant(choice)

	for _, call := range choice.ToolCalls {
		result, err := l.executeToolCall(ctx, run, call)
		if err != nil {
			if !l.isRecoverableToolError(err) {
				return err
			}

			result = err.Error()
		}

		run.appendToolResponse(call, result)
	}

	return nil
}

func (*OpenAI) isRecoverableToolError(err error) bool {
	return errors.Is(err, domain.ErrEmptyCommitMessage) ||
		errors.Is(err, domain.ErrCommitMessageTooLong) ||
		errors.Is(err, domain.ErrCommitMessageControl) ||
		errors.Is(err, domain.ErrCommitMessageNUL) ||
		errors.Is(err, domain.ErrCommitMessageANSI) ||
		errors.Is(err, domain.ErrForbiddenCommitTrailer) ||
		errors.Is(err, domain.ErrCommitAlreadyUpdated) ||
		errors.Is(err, domain.ErrCommitNotUpdated) ||
		errors.Is(err, domain.ErrUnknownTool) ||
		errors.Is(err, domain.ErrToolArguments)
}

func (l *OpenAI) executeToolCall(ctx context.Context, run *agentRun, call llms.ToolCall) (string, error) {
	if call.FunctionCall == nil {
		return "", fmt.Errorf("%w: %s", domain.ErrToolCallWithoutFunction, call.ID)
	}

	name := call.FunctionCall.Name
	l.log.Info("Calling tool", zap.String("tool", name))

	var result string
	var err error

	switch name {
	case toolGetCurrentBranch:
		result, err = l.getCurrentBranch(ctx)
	case toolLoadDiff:
		result, err = l.loadDiff(ctx, run)
	case toolUpdateFirstCommit:
		result, err = l.updateFirstCommit(ctx, run, call.FunctionCall.Arguments)
	case toolVerifyFirstCommit:
		result, err = l.verifyFirstCommit(ctx, run)
	default:
		err = fmt.Errorf("%w: %s", domain.ErrUnknownTool, name)
	}

	if err != nil {
		l.log.Warn("Tool failed", zap.String("tool", name), zap.Error(err))

		return "", err
	}

	l.log.Info("Tool completed", zap.String("tool", name))

	return result, nil
}

func (l *OpenAI) getCurrentBranch(ctx context.Context) (string, error) {
	res, err := l.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return wrapUntrustedData("branch", res.Value), nil
}

func (l *OpenAI) loadDiff(ctx context.Context, run *agentRun) (string, error) {
	if run.diffLoaded {
		return diffAlreadyLoaded, nil
	}

	res, err := l.git.LoadDiff(ctx, &ports.LoadDiffParams{
		BaseBranch: l.options.baseBranch,
		MaxBytes:   l.options.maxDiffBytes,
	})
	if err != nil {
		return "", fmt.Errorf("load diff: %w", err)
	}

	run.diffLoaded = true

	return wrapUntrustedData("diff", redactSecrets(res.Diff)), nil
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

	result, err := l.applyFirstCommit(ctx, input.CommitMessage)
	if err != nil {
		return "", err
	}

	run.updated = true
	run.message = input.CommitMessage
	run.backupRef = result.BackupRef

	return "First commit updated", nil
}

func (l *OpenAI) ensureNotBaseBranch(ctx context.Context) error {
	current, err := l.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	if current.Value == l.options.baseBranch {
		return fmt.Errorf("%w: %q", domain.ErrOnBaseBranch, current.Value)
	}

	return nil
}

func (l *OpenAI) applyFirstCommit(ctx context.Context, message string) (*ports.UpdateFirstCommitResult, error) {
	result, err := l.git.UpdateFirstCommit(ctx, &ports.UpdateFirstCommitParams{
		CommitMessage: message,
		BaseBranch:    l.options.baseBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("update first commit: %w", err)
	}

	return result, nil
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

	res, err := l.git.FirstCommitMessage(ctx, &ports.FirstCommitMessageParams{BaseBranch: l.options.baseBranch})
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

func (l *OpenAI) generateContent(
	ctx context.Context,
	messages []llms.MessageContent,
	tools []llms.Tool,
) (*llms.ContentResponse, int, error) {
	call := &contentCall{
		ctx:         ctx,
		log:         l.log,
		client:      l.client,
		messages:    messages,
		tools:       tools,
		retries:     l.options.retry.Retries,
		retryStatus: l.options.retryStatus,
	}

	resp, err := retry.RetryWithResult(ctx, call.generate,
		retry.WithRetries(l.options.retry.Retries),
		retry.WithInitInterval(l.options.retry.InitInterval),
		retry.WithMaxInterval(l.options.retry.MaxInterval),
	)
	if err != nil {
		return nil, call.attempt, fmt.Errorf("generate content: %w", err)
	}

	return resp, call.attempt, nil
}

func (c *contentCall) generate() (*llms.ContentResponse, error) {
	c.attempt++

	resp, err := c.client.GenerateContent(c.ctx, c.messages, llms.WithTools(c.tools))
	if err != nil {
		if c.ctx.Err() == nil && isRequestTimeout(err) {
			err = &retry.TimeoutError{Err: err}
		}

		err = c.withHTTPStatus(err)

		c.log.Warn("Generate content",
			zap.Int("attempt", c.attempt),
			zap.Uint64("max_attempts", c.retries+1),
			zap.Error(err))
	}

	return resp, err
}

func isRequestTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error

	return errors.As(err, &netErr) && netErr.Timeout()
}

func (c *contentCall) withHTTPStatus(err error) error {
	if c.retryStatus == nil {
		return err
	}

	statusCode, retryAfter := c.retryStatus.Take()
	if statusCode < http.StatusBadRequest {
		return err
	}

	return &retry.HTTPError{
		StatusCode: statusCode,
		RetryAfter: retryAfter,
		Err:        err,
	}
}

func (r *agentRun) firstChoice(resp *llms.ContentResponse) (*llms.ContentChoice, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return nil, domain.ErrNoResponseChoices
	}

	return resp.Choices[0], nil
}

func (r *agentRun) addUsage(choice *llms.ContentChoice) {
	if choice == nil {
		return
	}

	prompt, _ := choice.GenerationInfo["PromptTokens"].(int)
	completion, _ := choice.GenerationInfo["CompletionTokens"].(int)
	total, _ := choice.GenerationInfo["TotalTokens"].(int)

	r.promptTokens += prompt
	r.completionTokens += completion
	r.totalTokens += total
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

	return &ports.GenCommitDescriptionResult{
		Value:            run.message,
		BackupRef:        run.backupRef,
		PromptTokens:     run.promptTokens,
		CompletionTokens: run.completionTokens,
		TotalTokens:      run.totalTokens,
		Requests:         run.requests,
	}, nil
}

func (a *updateFirstCommitArgs) parse(arguments string) error {
	if err := json.Unmarshal([]byte(arguments), a); err != nil {
		return fmt.Errorf("%w: parse update first commit args: %w", domain.ErrToolArguments, err)
	}

	return validateCommitMessage(a.CommitMessage)
}

func wrapUntrustedData(kind, value string) string {
	value = untrustedTagPattern.ReplaceAllString(value, escapedUntrustedTag)

	return fmt.Sprintf("%s\nkind: %s\n%s\n%s", untrustedDataBegin, kind, value, untrustedDataEnd)
}

func validateCommitMessage(message string) error {
	if message == "" {
		return domain.ErrEmptyCommitMessage
	}

	if len(message) > maxCommitMessageBytes {
		return fmt.Errorf("%w: %d > %d bytes",
			domain.ErrCommitMessageTooLong, len(message), maxCommitMessageBytes)
	}

	for offset, r := range message {
		switch {
		case r == 0:
			return fmt.Errorf("%w at byte %d", domain.ErrCommitMessageNUL, offset)
		case r == '\n' || r == '\t':
		case r == 0x1b || (r >= 0x80 && r <= 0x9f):
			return fmt.Errorf("%w at byte %d", domain.ErrCommitMessageANSI, offset)
		case r < 0x20 || r == 0x7f:
			return fmt.Errorf("%w at byte %d", domain.ErrCommitMessageControl, offset)
		}
	}

	for _, line := range strings.Split(message, "\n") {
		if trailer := forbiddenTrailer(line); trailer != "" {
			return fmt.Errorf("%w: %q", domain.ErrForbiddenCommitTrailer, trailer)
		}
	}

	return nil
}

func forbiddenTrailer(line string) string {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "[skip ci]" {
		return "[skip ci]"
	}

	key, _, ok := strings.Cut(lower, ":")
	if !ok {
		return ""
	}

	switch strings.TrimSpace(key) {
	case "signed-off-by", "co-authored-by":
		return strings.TrimSpace(key)
	default:
		return ""
	}
}
