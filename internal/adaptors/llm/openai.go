package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
	"go.uber.org/zap"
)

type toolHandler func(ctx context.Context, args string) (string, error)

type OpenAI struct {
	log    *zap.Logger
	client *openai.LLM
	rules  string
	git    ports.Git
	tools  map[string]toolHandler
}

func NewOpenAI(log *zap.Logger, client *openai.LLM, rules string, git ports.Git) *OpenAI {
	entiy := &OpenAI{
		log:    log,
		client: client,
		rules:  rules,
		git:    git,
	}

	entiy.tools = map[string]toolHandler{
		"get_current_branch": entiy.getCurrentBranch,
	}

	return entiy
}

func (l *OpenAI) getCurrentBranch(ctx context.Context, _ string) (string, error) {
	res, err := l.git.CurrentBranch(ctx, &ports.CurrentBranchParams{})
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	return res.Value, nil
}

func (l *OpenAI) GenCommitDescription(ctx context.Context, params *ports.GenCommitDescriptionParams) (*ports.GenCommitDescriptionResult, error) {
	parts := []string{systemPrompt}
	if l.rules != "" {
		parts = append(parts, l.rules)
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, parts...),
		llms.TextParts(llms.ChatMessageTypeHuman, params.Diff),
	}

	tools := Tools()

	resp, err := l.client.GenerateContent(ctx, messages, llms.WithTools(tools))
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	for len(resp.Choices) > 0 && len(resp.Choices[0].ToolCalls) > 0 {
		assistantParts := []llms.ContentPart{}
		if resp.Choices[0].Content != "" {
			assistantParts = append(assistantParts, llms.TextPart(resp.Choices[0].Content))
		}

		for _, tc := range resp.Choices[0].ToolCalls {
			assistantParts = append(assistantParts, tc)
		}

		messages = append(messages, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: assistantParts,
		})

		for _, tc := range resp.Choices[0].ToolCalls {
			if tc.FunctionCall == nil {
				continue
			}

			l.log.Info("Сalling tool", zap.String("tool", tc.FunctionCall.Name))

			handler, ok := l.tools[tc.FunctionCall.Name]
			if !ok {
				return nil, fmt.Errorf("unknown tool: %s", tc.FunctionCall.Name)
			}

			result, err := handler(ctx, tc.FunctionCall.Arguments)
			if err != nil {
				return nil, fmt.Errorf("execute tool %s: %w", tc.FunctionCall.Name, err)
			}

			messages = append(messages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: tc.ID,
						Name:       tc.FunctionCall.Name,
						Content:    result,
					},
				},
			})
		}

		resp, err = l.client.GenerateContent(ctx, messages, llms.WithTools(tools))
		if err != nil {
			return nil, fmt.Errorf("generate content: %w", err)
		}
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%w", domain.ErrNoResponseChoices)
	}

	return &ports.GenCommitDescriptionResult{Value: strings.TrimSpace(resp.Choices[0].Content)}, nil
}
