package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultGeminiModel = "gemini-2.5-flash"
	geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"
)

// geminiProvider implements Provider using the Google Gemini REST API.
type geminiProvider struct {
	apiKey string
	model string
	client *http.Client
}

// NewGemini constructs a Gemini provider. apiKey must be non-empty.
func NewGemini(apiKey, model string) (Provider, error) {
	if apiKey == "" {
		return nil, &ErrProviderConfig{
			Provider: "gemini",
			Message:  "API key is required — set GEMINI_API_KEY env var",
		}
	}
	if model == "" {
		model = defaultGeminiModel
	}
	return &geminiProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}, nil
}

// Name implements Provider.
func (p *geminiProvider) Name() string {
	return "gemini"
}

// GenerateCommand implements Provider.
func (p *geminiProvider) GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error) {
	// Build the URL: https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=...
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIURL, p.model, p.apiKey)

	// Gemini combines system prompt + user message into a single contents array.
	// We prepend the system prompt as a "user" turn followed by a model ack, then add the real user query — this matches Gemini's recommended pattern
	// for injecting a system context without the system_instruction field (which is only available on Gemini 1.5+).
	type part struct {
		Text string `json:"text"`
	}

	type content struct {
		Role  string `json:"role,omitempty"` // omitempty = don't include if empty
		Parts []part `json:"parts"`
	}

	type genConfig struct {
		ResponseMimeType string `json:"responseMimeType"`
	}
	type reqBody struct {
		Contents []content `json:"contents"`
		SystemInstruction *content  `json:"system_instruction,omitempty"`
		GenerationConfig  genConfig `json:"generationConfig"`
	}

	userMessage := buildUserMessage(req)

	body := reqBody{
		SystemInstruction: &content{
			Parts: []part{{Text: req.SystemPrompt}},
		},
		Contents: []content{
			{
				Role: "user",
				Parts: []part{{Text: userMessage}},

			},		
		},
		GenerationConfig: genConfig{
			ResponseMimeType: "application/json",
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return CommandResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))

	if err != nil {
		return CommandResponse{}, fmt.Errorf("gemini: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CommandResponse{}, &ErrAPICall{Provider: "gemini", Message: err.Error()}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return CommandResponse{}, &ErrAPICall{Provider: "gemini", Message: "reading response body: " + err.Error()}
	}

	if resp.StatusCode != http.StatusOK {
		return CommandResponse{}, &ErrAPICall{
			Provider:   "gemini",
			StatusCode: resp.StatusCode,
			Message:    truncate(string(raw), 200),
		}
	}

	// Parse Gemini response envelope.
	var envelope struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || len(envelope.Candidates) == 0 {
		return CommandResponse{}, &ErrInvalidResponse{Provider: "gemini", Raw: truncate(string(raw), 200)}
	}

	parts := envelope.Candidates[0].Content.Parts
	if len(parts) == 0 {
		return CommandResponse{}, &ErrInvalidResponse{Provider: "gemini", Raw: "(empty candidates)"}
	}

	return parseAIResponse(parts[0].Text, req.OS, "gemini")
}

// buildUserMessage creates the text sent as the "user" turn to the AI.
func buildUserMessage(req CommandRequest) string {
	return fmt.Sprintf("OS: %s\nShell: %s\nQuery: %s", req.OS, req.Shell, req.Query)
}