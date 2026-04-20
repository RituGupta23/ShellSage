package prompts

// Enable //go:embed support (imported for side effects only).
import _ "embed"

// "//go:embed default.txt" - This is a compiler directive.
// Embed default.txt into this variable at compile time.
// After build, the file content is stored inside the binary (variable below (defaultPromptBytes)).
// File is no longer needed at runtime.

//go:embed default.txt
var defaultPromptBytes []byte

// Default returns the embedded prompt as a string.
// []byte → string conversion is cheap.
// Error is kept for future flexibility.
func Default() (string, error) {
	return string(defaultPromptBytes), nil
}