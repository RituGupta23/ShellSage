package provider

import (
	"context"
	"fmt"
)

type Provider interface {
	// returns provider name like "gemini", "claude"
	Name() string 
	// GenerateCommand calls the AI and returns structured commands.
	// ctx carries cancellation (Ctrl+C stops the API call).
	GenerateCommand(ctx context.Context, req CommandRequest) (CommandResponse, error)
}

type CommandRequest struct {
	Query string
	OS string
	Shell string
	SystemPrompt string
}

type CommandResponse struct {
	Primary OSCommand // the command for the user's detected OS
	Variants []OSCommand // the command for all OSes
	RiskLevel string // "low" | "medium" | "high"
	RiskReason string // why it's that risk level
}

type OSCommand struct {
	OS string
	Shell string
	Command string
}

// Custom Error Types

// ErrProviderConfig = misconfiguration 
type ErrProviderConfig struct {
	Provider string
	Message string
}

func (e *ErrProviderConfig) Error() string {
	return fmt.Sprintf("Provider %q: %s", e.Provider, e.Message)
}

// ErrAPICall = the AI API returned an error
type ErrAPICall struct {
	Provider string
	StatusCode int
	Message string
}

func (e *ErrAPICall) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("Provider %q: API error %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("Provider %q: %s", e.Provider, e.Message)
}

// ErrInvalidResponse = the AI returned text we can't parse as JSON
type ErrInvalidResponse struct {
	Provider string
	Raw string
}

func (e *ErrInvalidResponse) Error() string {
	return fmt.Sprintf("Provider %q: could not parse AI response. Raw: %s", e.Provider, e.Raw)
}

// FACTORY FUNCTION
func New(name, apiKey, model, baseURL string) (Provider, error) {
	switch name {
	case "gemini":
		return NewGemini(apiKey, model)
	default:
		return nil, fmt.Errorf("unknown provider %q — valid options: gemini", name)
	}
}