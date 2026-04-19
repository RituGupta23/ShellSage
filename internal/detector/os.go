package detector

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// holds detected OS and shell info
type OSInfo struct {
	OS string // macos | linux | windows
	Shell string // bash | zsh | powershell
}

// Return the current OS and shell
func Detect(osOverride string) OSInfo {
	detectedOS := detectOS()

	if osOverride != "" {
		if norm := normalizeOS(osOverride); norm != "" {
			detectedOS = norm
		}
	}

	return OSInfo{
		OS:    detectedOS,
		Shell: detectShell(detectedOS),
	}
}

func detectOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func normalizeOS(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "macos", "mac", "darwin", "osx":
		return "macos"
	case "linux":
		return "linux"
	case "windows", "win":
		return "windows"
	}
	return ""
}

func detectShell(osName string) string {
	switch osName {

	case "windows":
		return detectWindowsShell()

	default:
		shell := os.Getenv("SHELL")
		if shell != "" {
			base := filepath.Base(shell)

			switch base {
			case "zsh", "bash", "fish", "sh", "ksh", "dash":
				return base
			}
		}
		return defaultShell(osName)
	}
}

func detectWindowsShell() string {
	// Prefer PowerShell if detected
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}

	// Fallback to cmd
	if os.Getenv("ComSpec") != "" {
		return "cmd"
	}

	return "powershell"
}

func defaultShell(osName string) string {
	switch osName {
	case "macos":
		return "zsh"
	case "windows":
		return "powershell"
	default:
		return "bash"
	}
}

func OSLabel(osName string) string {
	switch osName {
	case "macos":
		return "macOS"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		return osName
	}
}