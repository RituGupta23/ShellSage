package cli

import (
	"fmt"
	"os"

	"github.com/RituGupta23/ShellSage/internal/config"
	"github.com/RituGupta23/ShellSage/internal/prompts"
)

// resolveSystemPrompt returns the system prompt string to use for the AI call.
// Priority:
//  1. User override file from config (cfg.Prompt.OverridePath)
//  2. Bundled default prompt (embedded via embed.FS in the prompts package)
func resolveSystemPrompt(cfg *config.Config) (string, error) {
	// User override takes priority.
	if cfg.Prompt.OverridePath != "" {
		data, err := os.ReadFile(cfg.Prompt.OverridePath)
		if err != nil {
			return "", fmt.Errorf("reading custom system prompt %q: %w", cfg.Prompt.OverridePath, err)
		}
		return string(data), nil
	}

	// Fall back to the embedded default prompt.
	data, err := prompts.Default()
	if err != nil {
		return "", fmt.Errorf("loading bundled system prompt: %w", err)
	}
	return data, nil
}

// bundledPrompt is kept for internal package use if needed elsewhere.
func bundledPrompt() (string, error) {
	return prompts.Default()
}
