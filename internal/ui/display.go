package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/RituGupta23/ShellSage/internal/detector"
	"github.com/RituGupta23/ShellSage/internal/provider"
	"github.com/RituGupta23/ShellSage/internal/risk"
)

// Renderer handles all terminal output and user interaction.
type Renderer struct {
	styles Styles
	stdout io.Writer
	stdin io.Reader
	noColor bool
}

// NewRenderer creates a Renderer. noColor disables ANSI codes.
func NewRenderer(noColor bool) *Renderer {
	return &Renderer{
		styles: NewStyles(noColor),
		stdout: os.Stdout,
		stdin: os.Stdin,
		noColor: noColor,
	}
}

// RenderResult is the outcome requested by the user.
type RenderResult int

const (
	// ResultDisplayOnly means the command was shown but not executed.
	ResultDisplayOnly RenderResult = iota
	// ResultExecuted means the command was run successfully.
	ResultExecuted
	// ResultAborted means the user declined to run the command.
	ResultAborted
)

// DisplayAndPrompt renders the AI response, classifies risk, prompts the user
// for confirmation if needed, and optionally executes the command.
//
// Parameters:
//   - resp: the AI-generated CommandResponse
//   - osInfo: the detected/overridden OS and shell
//   - wantRun: true when --run flag was set
//   - isDry: true when --dry flag was set (never executes)
func (r *Renderer) DisplayAndPrompt(
	ctx context.Context,
	resp provider.CommandResponse,
	osInfo detector.OSInfo,
	wantRun bool,
	isDry bool,
) (RenderResult, error) {
	// Classify risk — local classifier is the final arbiter; AI hint is advisory.
	classification := risk.OverrideFromAI(resp.Primary.Command, resp.RiskLevel, resp.RiskReason)

	r.renderVariants(resp, osInfo)
	r.renderRiskBanner(classification)

	// --dry: display only, never execute, even if high-risk.
	if isDry {
		r.printDim("--dry mode: command displayed but not executed.")
		return ResultDisplayOnly, nil
	}

	// In default (non-run) mode with low risk, still prompt.
	// wantRun just skips the "do you want me to run this?" intro question.
	shouldConfirm := wantRun || classification.Level != risk.Low

	if !shouldConfirm {
		// Low risk, display-only default mode.
		result, err := r.promptYN("Run macOS/Linux variant? [y/N]", false)
		if err != nil || !result {
			return ResultAborted, err
		}
	}

	// Determine which variant to run.
	cmdToRun, variantIdx, err := r.pickVariant(resp, osInfo)
	if err != nil {
		return ResultAborted, nil
	}
	_ = variantIdx

	// Risk-level–specific confirmation flow.
	confirmed, err := r.confirmForRisk(classification, cmdToRun)
	if err != nil {
		return ResultAborted, err
	}
	if !confirmed {
		return ResultAborted, nil
	}

	// Execute.
	if err := r.execute(ctx, cmdToRun, osInfo.Shell); err != nil {
		return ResultExecuted, fmt.Errorf("execution failed: %w", err)
	}
	fmt.Fprintln(r.stdout, r.styles.Success.Render("✓ Executed."))
	return ResultExecuted, nil
}

// renderVariants prints the command variant table.
func (r *Renderer) renderVariants(resp provider.CommandResponse, osInfo detector.OSInfo) {
	macosCmd, linuxCmd, windowsCmd := groupVariants(resp.Variants)

	mergeUnix := macosCmd == linuxCmd && macosCmd != ""

	fmt.Fprintln(r.stdout)
	if mergeUnix {
		r.printVariantRow("macOS / Linux", macosCmd)
		if windowsCmd != "" {
			r.printVariantRow("Windows (PowerShell)", windowsCmd)
		}
	} else {
		if macosCmd != "" {
			r.printVariantRow("macOS", macosCmd)
		}
		if linuxCmd != "" {
			r.printVariantRow("Linux", linuxCmd)
		}
		if windowsCmd != "" {
			r.printVariantRow("Windows (PowerShell)", windowsCmd)
		}
	}
	fmt.Fprintln(r.stdout)
}

func (r *Renderer) printVariantRow(label, cmd string) {
	styledLabel := r.styles.OSLabel.Render(fmt.Sprintf("  %-24s", label))
	arrow := r.styles.Arrow.Render("→")
	styledCmd := r.styles.CommandText.Render(cmd)
	fmt.Fprintf(r.stdout, "%s %s %s\n", styledLabel, arrow, styledCmd)
}

// renderRiskBanner prints the appropriate risk header.
func (r *Renderer) renderRiskBanner(c risk.Classification) {
	switch c.Level {
	case risk.High:
		fmt.Fprintln(r.stdout, r.styles.HighBanner.Render("  ✗ High risk — destructive command detected."))
	case risk.Medium:
		fmt.Fprintln(r.stdout, r.styles.MediumBanner.Render("  ⚠  Medium risk — this command mutates system state."))
	case risk.Low:
		// No banner for low-risk commands.
	}
}

// confirmForRisk implements the per-risk-level confirmation protocol.
func (r *Renderer) confirmForRisk(c risk.Classification, cmd string) (bool, error) {
	switch c.Level {
	case risk.High:
		// User must type the word "yes" exactly — "Y" alone is rejected.
		fmt.Fprintf(r.stdout, "  Type %s to proceed (\"Y\" alone is not accepted): ", r.styles.Bold.Render("yes"))
		answer, err := r.readLine()
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(answer) == "yes", nil

	case risk.Medium:
		ok, err := r.promptYN("  Proceed? [y/N]", false)
		return ok, err

	default: // Low
		ok, err := r.promptYN("  Run? [y/N]", false)
		return ok, err
	}
}

// pickVariant resolves which command string to run.
// If there are multiple meaningfully different variants, the user can type a number.
func (r *Renderer) pickVariant(resp provider.CommandResponse, osInfo detector.OSInfo) (string, int, error) {
	if len(resp.Variants) <= 1 {
		return resp.Primary.Command, 0, nil
	}

	// Check if we have genuinely different variants worth offering a choice for.
	macosCmd, linuxCmd, windowsCmd := groupVariants(resp.Variants)
	hasChoice := macosCmd != linuxCmd || windowsCmd != macosCmd

	if !hasChoice {
		return resp.Primary.Command, 0, nil
	}

	// Default to the detected OS variant; user can type 1/2/3 to override.
	defaultCmd := resp.Primary.Command

	// Build an ordered list for number selection.
	choices := buildChoices(macosCmd, linuxCmd, windowsCmd)
	if len(choices) <= 1 {
		return defaultCmd, 0, nil
	}

	fmt.Fprintln(r.stdout, r.styles.Dim.Render("  Which variant? (Enter for detected OS, or 1/2/3 for a specific one):"))
	for i, c := range choices {
		fmt.Fprintf(r.stdout, "    %d. %s\n", i+1, r.styles.Dim.Render(c.label+": "+c.cmd))
	}
	fmt.Fprint(r.stdout, "  Choice [default]: ")

	line, err := r.readLine()
	if err != nil {
		return defaultCmd, 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultCmd, 0, nil
	}

	idx := 0
	if _, err := fmt.Sscanf(line, "%d", &idx); err == nil {
		if idx >= 1 && idx <= len(choices) {
			return choices[idx-1].cmd, idx - 1, nil
		}
	}
	return defaultCmd, 0, nil
}

type variantChoice struct {
	label string
	cmd   string
}

func buildChoices(macosCmd, linuxCmd, windowsCmd string) []variantChoice {
	seen := map[string]bool{}
	var choices []variantChoice
	add := func(label, cmd string) {
		if cmd != "" && !seen[cmd] {
			seen[cmd] = true
			choices = append(choices, variantChoice{label: label, cmd: cmd})
		}
	}
	if macosCmd == linuxCmd {
		add("macOS / Linux", macosCmd)
	} else {
		add("macOS", macosCmd)
		add("Linux", linuxCmd)
	}
	add("Windows (PowerShell)", windowsCmd)
	return choices
}

// groupVariants extracts the three canonical OS variants from the AI response.
func groupVariants(variants []provider.OSCommand) (macosCmd, linuxCmd, windowsCmd string) {
	for _, v := range variants {
		switch strings.ToLower(v.OS) {
		case "macos", "darwin", "mac":
			macosCmd = v.Command
		case "linux":
			linuxCmd = v.Command
		case "windows", "win":
			windowsCmd = v.Command
		}
	}
	return
}

// execute runs the given command string in the user's detected shell.
func (r *Renderer) execute(ctx context.Context, cmdStr, shell string) error {
	var cmd *exec.Cmd

	switch strings.ToLower(shell) {
	case "powershell", "pwsh":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", cmdStr)
	case "cmd":
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	default:
		// zsh, bash, fish, sh — use -c flag.
		shellBin := shell
		if shellBin == "" {
			shellBin = "sh"
		}
		cmd = exec.CommandContext(ctx, shellBin, "-c", cmdStr)
	}

	cmd.Stdout = r.stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = r.stdin

	return cmd.Run()
}

// promptYN prints prompt and reads a Y/N answer. defaultYes controls the default
// when the user presses Enter without typing anything.
func (r *Renderer) promptYN(prompt string, defaultYes bool) (bool, error) {
	fmt.Fprint(r.stdout, prompt+" ")
	line, err := r.readLine()
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes, nil
	}
	return line == "y" || line == "yes", nil
}

func (r *Renderer) printDim(msg string) {
	fmt.Fprintln(r.stdout, r.styles.Dim.Render("  "+msg))
}

// readLine reads one line from stdin, stripping the newline.
func (r *Renderer) readLine() (string, error) {
	scanner := bufio.NewScanner(r.stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// EOF — treat as empty.
	return "", nil
}
