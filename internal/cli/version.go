package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version variables are injected at build time via -ldflags.
// Defaults are used when building without GoReleaser.
var (
	// Version is the semantic version string, e.g. "1.0.0".
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "none"
	// BuildDate is the ISO-8601 build timestamp.
	BuildDate = "unknown"
)

// versionCmd implements `sg version`.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print binary version and build info",
	RunE:  runVersion,
}

func runVersion(_ *cobra.Command, _ []string) error {
	fmt.Printf("ShellSage (sg)\n")
	fmt.Printf("  Version   : %s\n", Version)
	fmt.Printf("  Commit    : %s\n", Commit)
	fmt.Printf("  Built     : %s\n", BuildDate)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch   : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return nil
}
