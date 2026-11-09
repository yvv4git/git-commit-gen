package llm

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/llms"

	"github.com/yvv4git/git-commit-gen/internal/domain"
	"github.com/yvv4git/git-commit-gen/internal/ports"
)

const checkProviderPrompt = "Who are you?"

func (l *OpenAI) CheckProvider(ctx context.Context) (*ports.ProviderCheckResult, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, checkProviderPrompt),
	}

	resp, _, err := l.generateContent(ctx, messages, nil)
	if err != nil {
		return nil, err
	}

	if resp == nil || len(resp.Choices) == 0 {
		return nil, domain.ErrNoResponseChoices
	}

	return &ports.ProviderCheckResult{Value: strings.TrimSpace(resp.Choices[0].Content)}, nil
}
