package cli

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shellsage/sg/internal/config"
	"github.com/shellsage/sg/internal/provider"
)

// configCmd is the parent `sg config` command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ShellSage configuration",
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

// configInitCmd implements `sg config init` — the interactive setup wizard.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard — writes ~/.shellsage/config.toml",
	RunE:  runConfigInit,
}

// configShowCmd implements `sg config show`.
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current resolved config (API key redacted)",
	RunE:  runConfigShow,
}

func runConfigInit(cmd *cobra.Command, _ []string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\nShellSage Configuration Wizard")
	fmt.Println(strings.Repeat("─", 40))

	// 1. Provider selection.
	providerName := prompt(scanner, "AI provider [claude/openai/ollama/gemini] (default: claude): ", "claude")
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	if providerName != "claude" && providerName != "openai" && providerName != "ollama" && providerName != "gemini" {
		fmt.Fprintf(os.Stderr, "Unknown provider %q — defaulting to claude\n", providerName)
		providerName = "claude"
	}

	// 2. Model.
	defaultModel := defaultModelForProvider(providerName)
	model := prompt(scanner, fmt.Sprintf("Model (default: %s): ", defaultModel), defaultModel)
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}

	// 3. API key (skip for ollama).
	apiKey := ""
	baseURL := ""
	if providerName == "ollama" {
		baseURL = prompt(scanner, "Ollama base URL (default: http://localhost:11434): ", "http://localhost:11434")
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
	} else {
		envVar := apiKeyEnvVar(providerName)
		hint := ""
		if existing := os.Getenv(envVar); existing != "" {
			hint = fmt.Sprintf(" (detected %s in environment — press Enter to use it)", envVar)
		}
		apiKey = prompt(scanner, fmt.Sprintf("API key%s: ", hint), os.Getenv(envVar))
		apiKey = strings.TrimSpace(apiKey)
	}

	// 4. Default mode.
	modeRaw := prompt(scanner, "Default mode [dry/run] (default: dry): ", "dry")
	modeRaw = strings.ToLower(strings.TrimSpace(modeRaw))
	if modeRaw != "dry" && modeRaw != "run" {
		modeRaw = "dry"
	}

	// 5. Validate the API key with a cheap API call (skip for ollama).
	if providerName != "ollama" && apiKey != "" {
		fmt.Printf("Validating API key with %s...", providerName)
		if err := validateAPIKey(cmd.Context(), providerName, apiKey, model); err != nil {
			fmt.Printf(" failed (%v)\n", err)
			fmt.Println("Config will be written, but the API key may be invalid.")
		} else {
			fmt.Println(" OK")
		}
	}

	// 6. Write config.
	configDir, err := config.ConfigDirPath()
	if err != nil {
		return fmt.Errorf("resolving config dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("creating config dir %q: %w", configDir, err)
	}

	cfgPath := filepath.Join(configDir, "config.toml")
	contents := config.DefaultConfigTOML(providerName, model, apiKey, baseURL)
	if err := os.WriteFile(cfgPath, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("writing config file %q: %w", cfgPath, err)
	}

	fmt.Printf("\nConfig written to %s\n", cfgPath)
	fmt.Printf("You can now run: sg \"your query here\"\n\n")
	return nil
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfgPath, _ := config.ConfigFilePath()

	fmt.Printf("\nResolved ShellSage Configuration\n")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Config file : %s\n", cfgPath)
	fmt.Printf("\n[provider]\n")
	fmt.Printf("  name      = %q\n", cfg.Provider.Name)
	fmt.Printf("  model     = %q\n", cfg.Provider.Model)
	fmt.Printf("  api_key   = %q\n", config.RedactedAPIKey(cfg.Provider.APIKey))
	if cfg.Provider.BaseURL != "" {
		fmt.Printf("  base_url  = %q\n", cfg.Provider.BaseURL)
	}
	fmt.Printf("\n[behavior]\n")
	fmt.Printf("  default_mode  = %q\n", cfg.Behavior.DefaultMode)
	fmt.Printf("  confirm_risky = %v\n", cfg.Behavior.ConfirmRisky)
	fmt.Printf("  os_override   = %q\n", cfg.Behavior.OSOverride)
	fmt.Printf("  no_color      = %v\n", cfg.Behavior.NoColor)
	fmt.Printf("\n[prompt]\n")
	if cfg.Prompt.OverridePath != "" {
		fmt.Printf("  override_path = %q\n", cfg.Prompt.OverridePath)
	} else {
		fmt.Printf("  override_path = (using bundled default)\n")
	}
	fmt.Println()
	return nil
}

// prompt prints the prompt string, reads a line from scanner, and returns
// the defaultVal if the user enters nothing.
func prompt(scanner *bufio.Scanner, promptStr, defaultVal string) string {
	fmt.Print(promptStr)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return defaultVal
}

func defaultModelForProvider(p string) string {
	switch p {
	case "claude":
		return "claude-sonnet-4-20250514"
	case "openai":
		return "gpt-4o"
	case "ollama":
		return "llama3"
	case "gemini":
		return "gemini-2.5-flash"
	default:
		return ""
	}
}

func apiKeyEnvVar(p string) string {
	switch p {
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	default:
		return "SHELLSAGE_API_KEY"
	}
}

// validateAPIKey makes a minimal API call to confirm the key is valid.
// It builds a real provider and issues a very short query.
func validateAPIKey(ctx context.Context, providerName, apiKey, model string) error {
	// Use a short timeout for validation only.
	ctx2, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Ollama doesn't require an API key — skip.
	if providerName == "ollama" {
		return pingOllama(ctx2, "http://localhost:11434")
	}

	prov, err := provider.New(providerName, apiKey, model, "")
	if err != nil {
		return err
	}

	// We use a trivial query so the token cost is negligible.
	bundled, _ := bundledPrompt()
	_, err = prov.GenerateCommand(ctx2, provider.CommandRequest{
		Query:        "echo hello",
		OS:           "linux",
		Shell:        "bash",
		SystemPrompt: bundled,
	})
	return err
}

// pingOllama does a simple GET to check Ollama is reachable.
func pingOllama(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama is not reachable at %s: %w", baseURL, err)
	}
	resp.Body.Close()
	return nil
}
