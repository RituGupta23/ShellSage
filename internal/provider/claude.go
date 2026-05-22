package provider

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const defaultClaudeModel = "claude-sonnet-4-20250514"

// claudeProvider implements Provider using the Anthropic Claude API.
type claudeProvider struct {
	client *anthropic.Client
	model  string
}

// NewClaude constructs a Claude provider. apiKey must be non-empty.
func NewClaude(apiKey, model string) (Provider, error) {
	if apiKey == "" {
		return nil, &ErrProviderConfig{
			Provider: "claude",
			Message:  "API key is required — set ANTHROPIC_API_KEY or configure provider.api_key in ~/.shellsage/config.toml",
		}
	}
	if model == "" {
		model = defaultClaudeModel
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &claudeProvider{client: client, model: model}, nil
}

// Name implements Provider.
func (p *claudeProvider) Name() string { return "claude" }

// GenerateCommand implements Provider.
func (p *claudeProvider) GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error) {
	userMessage := buildUserMessage(req)

	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model(p.model)),
		MaxTokens: anthropic.F(int64(1024)),
		System: anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(req.SystemPrompt),
		}),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		}),
	})
	if err != nil {
		return CommandResponse{}, &ErrAPICall{
			Provider: "claude",
			Message:  fmt.Sprintf("messages.new failed: %v", err),
		}
	}

	if len(msg.Content) == 0 {
		return CommandResponse{}, &ErrInvalidResponse{
			Provider: "claude",
			Raw:      "(empty response)",
		}
	}

	raw := msg.Content[0].Text
	return parseAIResponse(raw, req.OS, "claude")
}

// buildUserMessage constructs the user-turn message sent to the AI.
func buildUserMessage(req CommandRequest) string {
	return fmt.Sprintf(
		"OS: %s\nShell: %s\nQuery: %s",
		req.OS, req.Shell, req.Query,
	)
}
