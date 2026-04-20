// Package cli wires together all Cobra commands and global flags.
// Why internal
// Any package inside internal/ is private to this module
// No one outside github.com/shellsage/sg can import it.
// Go's way to implement => " This is an implementation detail, not public API."

package cli

import (
	"context"
	"fmt"
	"strings"
	"github.com/spf13/cobra" // THIRD-PARTY import
	"github.com/shellsage/sg/internal/detector"
	"github.com/shellsage/sg/internal/prompts"
)

// globalFlags holds parsed value from CLI flags
type globalFlags struct {
	run bool // --run : execute the command
	dry bool // --dry : display the command
	osOverride string // --osOverride : override detected os
	prov string // --provider: which AI to use
	model string // --model : which AI model
	noColor bool // --noColor : disable colors
}
	
var gf globalFlags // this variable is accessible throughout the package (but not exported as started with lowercase)

// cobra.Command is a struct that describes one CLI command.

var rootCmd = &cobra.Command {
	Use: "sg [query]", // how the command is invoked
	Short: "ShellSage - translate plain English into shell commands", // what the command do
	Long: `ShellSage uses AI to translate plain English into shell commands.
	Examples:
    sg "find all log files modified in the last 3 days"
    sg --run "kill the process using port 3000"
    sg --provider gemini "compress this folder" `,  // more detailed description
	Args: cobra.MinimumNArgs(1), // validation rule - atleast 1 argument required
	RunE: runRoot, // function to execute when command is run
	SilenceUsage: true, 
	SilenceErrors: true, 
}

// Execute is the entry point called by main.go.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// init() is a special Go function. It runs AUTOMATICALLY before main().
// Every package can have init() functions. We use it to register flags with Cobra.
func init() {
	rootCmd.PersistentFlags().BoolVar(&gf.run, "run", false, "Execute the generated command immediately")
	rootCmd.PersistentFlags().BoolVar(&gf.dry, "dry", false, "Display only, never execute")
	rootCmd.PersistentFlags().StringVar(&gf.osOverride, "os", "", "Override detected OS: macos | linux | windows")
	rootCmd.PersistentFlags().StringVar(&gf.prov, "provider", "", "AI provider: claude | openai | ollama | gemini")
	rootCmd.PersistentFlags().StringVar(&gf.model, "model", "", "Override AI model name")
	rootCmd.PersistentFlags().BoolVar(&gf.noColor, "no-color", false, "Disable colored output")
}

// runRoot is called when the user types: sg "some query"
// cmd = the Cobra command object, args = the positional arguments
func runRoot(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	osInfo := detector.Detect(gf.osOverride)

	fmt.Printf("Query: %s\n", query)
	fmt.Printf("OS: %s\n", osInfo.OS)
	fmt.Printf("Shell: %s\n", osInfo.Shell)

	systemPrompt, _ := prompts.Default()
	fmt.Printf("Prompt loaded: %d characters\n", len(systemPrompt))

	return nil
}