// Every go file starts with package declaration
// package main - It tells Go "It is an executable program" not a library
// main function in the package main is the entry point (where our program starts running)

package main

// import is used to import packages
// fmt - Used to format and print output
// os - Used to access operating system functionality
import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shellsage/sg/internal/cli"
)

func main() {
	// signal.NotifyContext creates a context that auto-cancels on Ctrl+C.
	// os.Interrupt = Ctrl+C
	// syscall.SIGTERM = `kill` command (used by Docker, Kubernetes)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// cli.Execute starts the Cobra command parser.
	// If it returns an error, print it and exit with code 1.
	if err := cli.Execute(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "sg: %v\n", err)
		os.Exit(1)
	}
}