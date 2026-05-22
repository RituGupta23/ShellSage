package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultOllamaBaseURL = "http://localhost:11434"
	defaultOllamaModel   = "llama3"
	ollamaRequestTimeout = 120 * time.Second
)

// ollamaProvider implements Provider using the Ollama local API via stdlib net/http.
type ollamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllama constructs an Ollama provider. baseURL defaults to http://localhost:11434.
func NewOllama(baseURL, model string) (Provider, error) {
	if baseURL == "" {
		baseURL = defaultOllamaBaseURL
	}
	// Remove trailing slash for consistent URL building.
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = defaultOllamaModel
	}
	return &ollamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: ollamaRequestTimeout},
	}, nil
}

// Name implements Provider.
func (p *ollamaProvider) Name() string { return "ollama" }

// ollamaChatRequest is the JSON body sent to the Ollama /api/chat endpoint.
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatResponse is the JSON body returned by the Ollama /api/chat endpoint.
type ollamaChatResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error"`
}

// GenerateCommand implements Provider.
func (p *ollamaProvider) GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error) {
	userMessage := buildUserMessage(req)

	body := ollamaChatRequest{
		Model: p.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: userMessage},
		},
		Stream: false,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return CommandResponse{}, fmt.Errorf("ollama: marshalling request: %w", err)
	}

	url := p.baseURL + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return CommandResponse{}, fmt.Errorf("ollama: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CommandResponse{}, &ErrAPICall{
			Provider: "ollama",
			Message:  fmt.Sprintf("HTTP request failed: %v — is Ollama running at %s?", err, p.baseURL),
		}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return CommandResponse{}, fmt.Errorf("ollama: reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CommandResponse{}, &ErrAPICall{
			Provider:   "ollama",
			StatusCode: resp.StatusCode,
			Message:    truncate(string(respBytes), 200),
		}
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBytes, &ollamaResp); err != nil {
		return CommandResponse{}, &ErrInvalidResponse{
			Provider: "ollama",
			Raw:      truncate(string(respBytes), 200),
		}
	}

	if ollamaResp.Error != "" {
		return CommandResponse{}, &ErrAPICall{
			Provider: "ollama",
			Message:  ollamaResp.Error,
		}
	}

	return parseAIResponse(ollamaResp.Message.Content, req.OS, "ollama")
}
