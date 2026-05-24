// Package cli wires together all Cobra commands and global flags for ShellSage.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RituGupta23/ShellSage/internal/config"
	"github.com/RituGupta23/ShellSage/internal/detector"
	"github.com/RituGupta23/ShellSage/internal/provider"
	"github.com/RituGupta23/ShellSage/internal/ui"
)

// globalFlags holds values parsed from persistent/root-level flags.
type globalFlags struct {
	run        bool
	dry        bool
	osOverride string
	prov       string
	model      string
	noColor    bool
}

var gf globalFlags

// rootCmd is the top-level Cobra command — invoked as `sg "query"`.
var rootCmd = &cobra.Command{
	Use:   "sg [query]",
	Short: "ShellSage — translate plain English into shell commands",
	Long: `ShellSage uses AI to translate plain English into shell commands.

Examples:
  sg "find all log files modified in the last 3 days"
  sg --run "kill the process using port 3000"
  sg --dry "delete all node_modules folders recursively"
  sg --os linux "compress the project folder into a zip"`,
	Args:              cobra.MinimumNArgs(1),
	RunE:              runRoot,
	SilenceUsage:      true,
	SilenceErrors:     true,
	DisableFlagParsing: false,
}

// Execute is the entry point called by main.go.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&gf.run, "run", false,
		"Execute the generated command immediately after confirmation")
	rootCmd.PersistentFlags().BoolVar(&gf.dry, "dry", false,
		"Display only, never execute (overrides config default)")
	rootCmd.PersistentFlags().StringVar(&gf.osOverride, "os", "",
		"Override detected OS: macos | linux | windows")
	rootCmd.PersistentFlags().StringVar(&gf.prov, "provider", "",
		"Override config provider: claude | openai | ollama | gemini")
	rootCmd.PersistentFlags().StringVar(&gf.model, "model", "",
		"Override model (e.g. gemini-1.5-flash, gpt-4o-mini, claude-haiku-4-20250514)")
	rootCmd.PersistentFlags().BoolVar(&gf.noColor, "no-color", false,
		"Disable lipgloss styling (plain text output)")

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// runRoot implements the main `sg "query"` flow.
func runRoot(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Load config.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply flag overrides to cfg.
	applyFlagOverrides(cfg)

	// Detect OS and shell.
	osInfo := detector.Detect(cfg.Behavior.OSOverride)

	// Resolve the system prompt.
	systemPrompt, err := resolveSystemPrompt(cfg)
	if err != nil {
		return fmt.Errorf("loading system prompt: %w", err)
	}

	// Build the provider.
	prov, err := provider.New(cfg.Provider.Name, cfg.Provider.APIKey, cfg.Provider.Model, cfg.Provider.BaseURL)
	if err != nil {
		return fmt.Errorf("initialising provider: %w", err)
	}

	// Call AI.
	req := provider.CommandRequest{
		Query:        query,
		OS:           osInfo.OS,
		Shell:        osInfo.Shell,
		SystemPrompt: systemPrompt,
	}
	resp, err := prov.GenerateCommand(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("generating command: %w", err)
	}

	// Render result and handle user interaction.
	isDry := gf.dry || cfg.Behavior.DefaultMode == "dry" && !gf.run
	renderer := ui.NewRenderer(cfg.Behavior.NoColor)
	_, err = renderer.DisplayAndPrompt(cmd.Context(), resp, osInfo, gf.run, isDry)
	return err
}

// applyFlagOverrides copies CLI flag values into the resolved config, giving flags
// the highest priority (flags > env > config file > defaults).
func applyFlagOverrides(cfg *config.Config) {
	if gf.osOverride != "" {
		cfg.Behavior.OSOverride = gf.osOverride
	}
	if gf.prov != "" {
		prevProvider := cfg.Provider.Name
		cfg.Provider.Name = gf.prov
		// Re-resolve the API key for the newly selected provider so that
		// provider-specific env vars (e.g. GEMINI_API_KEY) are picked up
		// even when the flag overrides the default provider after Load().
		cfg.Provider.APIKey = config.ResolveAPIKey(cfg.Provider.Name, cfg.Provider.APIKey)
		// Reset model to the new provider's default when the provider changed,
		// so a Claude model name isn't accidentally sent to Gemini/OpenAI.
		if prevProvider != cfg.Provider.Name && cfg.Provider.Model == config.DefaultModel {
			cfg.Provider.Model = defaultModelForProvider(cfg.Provider.Name)
		}
	}
	// --model flag takes highest priority and always wins.
	if gf.model != "" {
		cfg.Provider.Model = gf.model
	}
	if gf.noColor {
		cfg.Behavior.NoColor = true
	}
	// Check the NO_COLOR env var (https://no-color.org).
	if os.Getenv("NO_COLOR") != "" {
		cfg.Behavior.NoColor = true
	}
}
