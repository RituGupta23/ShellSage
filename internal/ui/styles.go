// Package ui provides terminal rendering and user interaction for ShellSage.
package ui

import "github.com/charmbracelet/lipgloss"

// Palette defines the semantic color set used throughout the UI.
// All colors are chosen to work on both dark and light terminals.
var (
	colorHighRisk   = lipgloss.Color("#FF4444") // bright red
	colorMediumRisk = lipgloss.Color("#FFA500") // amber
	colorLowRisk    = lipgloss.Color("#00CC88") // teal-green
	colorCommand    = lipgloss.Color("#7DCFFF") // sky blue — code / commands
	colorLabel      = lipgloss.Color("#BB9AF7") // soft purple — OS labels
	colorSuccess    = lipgloss.Color("#9ECE6A") // green — success messages
	colorDim        = lipgloss.Color("#565F89") // muted — secondary text
	colorArrow      = lipgloss.Color("#7AA2F7") // blue — arrows/bullets
)

// Styles holds all pre-built lipgloss styles. Build once per run to avoid
// repeated allocations; set NoColor = true before calling NewStyles to disable.
type Styles struct {
	// Risk banners.
	HighBanner   lipgloss.Style
	MediumBanner lipgloss.Style
	LowBanner    lipgloss.Style

	// Command block.
	CommandText lipgloss.Style
	Arrow       lipgloss.Style
	OSLabel     lipgloss.Style

	// Feedback.
	Success lipgloss.Style
	Dim     lipgloss.Style
	Bold    lipgloss.Style
}

// NewStyles returns a Styles set. When noColor is true, all styles are stripped
// of ANSI so the output is plain text (respects --no-color and NO_COLOR env var).
func NewStyles(noColor bool) Styles {
	if noColor {
		return plainStyles()
	}
	return colorStyles()
}

func colorStyles() Styles {
	return Styles{
		HighBanner: lipgloss.NewStyle().
			Foreground(colorHighRisk).
			Bold(true),
		MediumBanner: lipgloss.NewStyle().
			Foreground(colorMediumRisk).
			Bold(true),
		LowBanner: lipgloss.NewStyle().
			Foreground(colorLowRisk),

		CommandText: lipgloss.NewStyle().
			Foreground(colorCommand),
		Arrow: lipgloss.NewStyle().
			Foreground(colorArrow).
			Bold(true),
		OSLabel: lipgloss.NewStyle().
			Foreground(colorLabel),

		Success: lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true),
		Dim: lipgloss.NewStyle().
			Foreground(colorDim),
		Bold: lipgloss.NewStyle().
			Bold(true),
	}
}

// plainStyles returns identity styles that render no ANSI codes.
func plainStyles() Styles {
	plain := lipgloss.NewStyle()
	return Styles{
		HighBanner:   plain,
		MediumBanner: plain,
		LowBanner:    plain,
		CommandText:  plain,
		Arrow:        plain,
		OSLabel:      plain,
		Success:      plain,
		Dim:          plain,
		Bold:         plain,
	}
}
