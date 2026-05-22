// Package risk provides pattern-based command risk classification.
package risk

import (
	"strings"
)

// ---- Types ----

type Level string

const (
	Low    Level = "low"
	Medium Level = "medium"
	High   Level = "high"
)

type Classification struct {
	Level  Level
	Reason string
}

// ---- Public API ----

func Classify(command string) Classification {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return Classification{Level: Low, Reason: "Empty command."}
	}

	parts := splitCommands(cmd)

	final := Classification{Level: Low, Reason: "No risky patterns detected."}

	for _, part := range parts {
		c := classifySingle(part)

		if riskOrdinal(c.Level) > riskOrdinal(final.Level) {
			final = c
		}
	}

	return final
}

// ---- Core Logic ----

func classifySingle(cmd string) Classification {
	tokens := tokenize(cmd)
	if len(tokens) == 0 {
		return Classification{Level: Low, Reason: "Empty command."}
	}

	base := tokens[0]
	flags := extractFlags(tokens)
	args := extractArgs(tokens)

	score := 0
	reasons := []string{}

	// ---- Context awareness ----

	if base == "docker" || base == "kubectl" {
		score += 1
		reasons = append(reasons, "Container/orchestration context.")
	}

	if base == "git" {
		score += 1
		reasons = append(reasons, "Version control operation.")
	}

	// ---- Command-specific logic ----

	switch base {

	case "rm":
		score += 3
		reasons = append(reasons, "File deletion command.")

		if hasFlag(flags, "r") || hasFlag(flags, "rf") {
			score += 3
			reasons = append(reasons, "Recursive deletion.")
		}

		if hasFlag(flags, "f") {
			score += 2
			reasons = append(reasons, "Force flag used.")
		}

		if targetsCriticalPath(args) {
			score += 5
			reasons = append(reasons, "Targets critical system path.")
		}

	case "dd", "mkfs":
		score += 8
		reasons = append(reasons, "Disk-level operation.")

	case "shutdown", "reboot", "halt":
		score += 6
		reasons = append(reasons, "System-level operation.")

	case "chmod":
		score += 2
		reasons = append(reasons, "Permission change.")

		if contains(args, "777") {
			score += 4
			reasons = append(reasons, "Overly permissive (777).")
		}

	case "chown":
		score += 2
		reasons = append(reasons, "Ownership change.")

	case "mv":
		score += 2
		reasons = append(reasons, "File move operation.")

	case "cp":
		if hasFlag(flags, "r") {
			score += 2
			reasons = append(reasons, "Recursive copy.")
		}

	case "git":
		if contains(args, "reset") || contains(args, "rebase") {
			score += 4
			reasons = append(reasons, "History rewrite.")
		}
		if contains(args, "--force") {
			score += 3
			reasons = append(reasons, "Force push.")
		}

	case "curl":
		if hasFlag(flags, "X") {
			method := getFlagValue(tokens, "-X")
			if method != "" && method != "GET" {
				score += 3
				reasons = append(reasons, "HTTP mutation request.")
			}
		}

	case "sed":
		if hasFlag(flags, "i") {
			score += 3
			reasons = append(reasons, "In-place file edit.")
		}
	}

	// ---- Generic heuristics ----

	if containsPipe(cmd) {
		score += 1
		reasons = append(reasons, "Piped command.")
	}

	// ---- Final classification ----

	level := scoreToLevel(score)

	return Classification{
		Level:  level,
		Reason: strings.Join(reasons, " "),
	}
}

// ---- Helpers ----

func tokenize(cmd string) []string {
	return strings.Fields(cmd)
}

func extractFlags(tokens []string) []string {
	var flags []string
	for _, t := range tokens {
		if strings.HasPrefix(t, "-") {
			flags = append(flags, strings.TrimLeft(t, "-"))
		}
	}
	return flags
}

func extractArgs(tokens []string) []string {
	var args []string
	for _, t := range tokens[1:] {
		if !strings.HasPrefix(t, "-") {
			args = append(args, t)
		}
	}
	return args
}

func hasFlag(flags []string, target string) bool {
	for _, f := range flags {
		if f == target || strings.Contains(f, target) {
			return true
		}
	}
	return false
}

func getFlagValue(tokens []string, flag string) string {
	for i, t := range tokens {
		if t == flag && i+1 < len(tokens) {
			return strings.ToUpper(tokens[i+1])
		}
	}
	return ""
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if strings.Contains(v, val) {
			return true
		}
	}
	return false
}

func containsPipe(cmd string) bool {
	return strings.Contains(cmd, "|")
}

func targetsCriticalPath(args []string) bool {
	for _, a := range args {
		if a == "/" || a == "/*" || a == "~" || strings.HasPrefix(a, "/") {
			return true
		}
	}
	return false
}

func splitCommands(cmd string) []string {
	return strings.FieldsFunc(cmd, func(r rune) bool {
		return r == ';' || r == '&'
	})
}

func scoreToLevel(score int) Level {
	switch {
	case score >= 7:
		return High
	case score >= 3:
		return Medium
	default:
		return Low
	}
}

func riskOrdinal(l Level) int {
	switch l {
	case High:
		return 2
	case Medium:
		return 1
	default:
		return 0
	}
}

func OverrideFromAI(command, aiLevel, aiReason string) Classification {
	local := Classify(command)

	var ai Classification
	switch strings.ToLower(strings.TrimSpace(aiLevel)) {
	case "high":
		ai = Classification{Level: High, Reason: aiReason}
	case "medium":
		ai = Classification{Level: Medium, Reason: aiReason}
	case "low":
		ai = Classification{Level: Low, Reason: aiReason}
	default:
		// Ignore invalid AI output
		return local
	}

	// Always pick the MORE dangerous one
	if riskOrdinal(local.Level) >= riskOrdinal(ai.Level) {
		return local
	}

	// AI detected something more dangerous → accept it
	return ai
}