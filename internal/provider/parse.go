// internal/provider/parse.go (temporary — we'll expand in Phase 6)
package provider

import (
	"encoding/json"
	"strings"
)

type aiResponseJSON struct {
	Variants []struct {
		OS      string `json:"os"`
		Shell   string `json:"shell"`
		Command string `json:"command"`
	} `json:"variants"`
	RiskLevel  string `json:"risk_level"`
	RiskReason string `json:"risk_reason"`
}

func parseAIResponse(raw, detectedOS, providerName string) (CommandResponse, error) {
	cleaned := strings.TrimSpace(raw)

	var parsed aiResponseJSON
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return CommandResponse{}, &ErrInvalidResponse{Provider: providerName, Raw: truncate(raw, 200)}
	}

	variants := make([]OSCommand, 0, len(parsed.Variants))
	var primary OSCommand
	for _, v := range parsed.Variants {
		cmd := OSCommand{OS: v.OS, Shell: v.Shell, Command: v.Command}
		variants = append(variants, cmd)
		if strings.EqualFold(v.OS, detectedOS) {
			primary = cmd
		}
	}
	if primary.Command == "" && len(variants) > 0 {
		primary = variants[0]
	}

	return CommandResponse{
		Primary: primary, Variants: variants,
		RiskLevel: parsed.RiskLevel, RiskReason: parsed.RiskReason,
	}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
