package provider

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const defaultOpenAIModel = "gpt-4o"

// openaiProvider implements Provider using the OpenAI API.
type openaiProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAI constructs an OpenAI provider. apiKey must be non-empty.
func NewOpenAI(apiKey, model string) (Provider, error) {
	if apiKey == "" {
		return nil, &ErrProviderConfig{
			Provider: "openai",
			Message:  "API key is required — set OPENAI_API_KEY or configure provider.api_key in ~/.shellsage/config.toml",
		}
	}
	if model == "" {
		model = defaultOpenAIModel
	}
	client := openai.NewClient(apiKey)
	return &openaiProvider{client: client, model: model}, nil
}

// Name implements Provider.
func (p *openaiProvider) Name() string { return "openai" }

// GenerateCommand implements Provider.
func (p *openaiProvider) GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error) {
	userMessage := buildUserMessage(req)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: req.SystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userMessage,
			},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return CommandResponse{}, &ErrAPICall{
			Provider: "openai",
			Message:  fmt.Sprintf("chat completion failed: %v", err),
		}
	}

	if len(resp.Choices) == 0 {
		return CommandResponse{}, &ErrInvalidResponse{
			Provider: "openai",
			Raw:      "(empty choices)",
		}
	}

	raw := resp.Choices[0].Message.Content
	return parseAIResponse(raw, req.OS, "openai")
}
