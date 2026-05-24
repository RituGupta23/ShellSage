package shellsage

import (
	"context"
	"fmt"
	"os"

	"github.com/shellsage/sg/internal/config"
	"github.com/shellsage/sg/internal/detector"
	"github.com/shellsage/sg/internal/provider"
	"github.com/shellsage/sg/internal/prompts"
)

type Config = config.Config
type ProviderConfig = config.ProviderConfig
type BehaviorConfig = config.BehaviorConfig
type PromptConfig = config.PromptConfig

type CommandRequest = provider.CommandRequest
type CommandResponse = provider.CommandResponse
type OSCommand = provider.OSCommand

type OSInfo = detector.OSInfo

// LoadConfig loads ShellSage configuration from ~/.shellsage/config.toml
// and environment variables, using the same resolution logic as the CLI.
func LoadConfig() (*Config, error) {
	return config.Load()
}

// RunQuery executes a natural language query using the loaded config.
// It returns the structured provider response, but does not display UI or execute shell commands.
func RunQuery(ctx context.Context, cfg *Config, query string) (CommandResponse, error) {
	systemPrompt, err := resolveSystemPrompt(cfg)
	if err != nil {
		return CommandResponse{}, err
	}

	osInfo := detector.Detect(cfg.Behavior.OSOverride)

	prov, err := provider.New(cfg.Provider.Name, cfg.Provider.APIKey, cfg.Provider.Model, cfg.Provider.BaseURL)
	if err != nil {
		return CommandResponse{}, fmt.Errorf("initialising provider: %w", err)
	}

	req := provider.CommandRequest{
		Query:        query,
		OS:           osInfo.OS,
		Shell:        osInfo.Shell,
		SystemPrompt: systemPrompt,
	}

	return prov.GenerateCommand(ctx, req)
}

func resolveSystemPrompt(cfg *Config) (string, error) {
	if cfg.Prompt.OverridePath != "" {
		data, err := os.ReadFile(cfg.Prompt.OverridePath)
		if err != nil {
			return "", fmt.Errorf("reading custom system prompt %q: %w", cfg.Prompt.OverridePath, err)
		}
		return string(data), nil
	}

	data, err := prompts.Default()
	if err != nil {
		return "", fmt.Errorf("loading bundled system prompt: %w", err)
	}
	return data, nil
}
