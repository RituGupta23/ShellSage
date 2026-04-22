package provider

import (
	"encoding/json"
	"regexp"
	"strings"
)

// aiResponseJSON is the expected JSON shape from the AI model.
type aiResponseJSON struct {
	Variants []struct {
		OS      string `json:"os"`
		Shell   string `json:"shell"`
		Command string `json:"command"`
	} `json:"variants"`
	RiskLevel  string `json:"risk_level"`
	RiskReason string `json:"risk_reason"`
}

// jsonBlockRe matches the outermost {...} JSON object in a string.
var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

// parseAIResponse extracts a CommandResponse from a raw AI text response.
// It strips any surrounding markdown code fences before attempting JSON decode.
func parseAIResponse(raw, detectedOS, providerName string) (CommandResponse, error) {
	cleaned := stripMarkdownFences(strings.TrimSpace(raw))

	var parsed aiResponseJSON
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// Fallback: extract the first {...} block in case the model injected stray characters outside the JSON
		extracted := jsonBlockRe.FindString(cleaned)
		if extracted == "" || json.Unmarshal([]byte(extracted), &parsed) != nil {
			return CommandResponse{}, &ErrInvalidResponse{
				Provider: providerName,
				Raw:      truncate(raw, 200),
			}
		}
	}

	variants := make([]OSCommand, 0, len(parsed.Variants))
	var primary OSCommand
	for _, v := range parsed.Variants {
		cmd := OSCommand{
			OS:      v.OS,
			Shell:   v.Shell,
			Command: v.Command,
		}
		variants = append(variants, cmd)
		if strings.EqualFold(v.OS, detectedOS) {
			primary = cmd
		}
	}

	// If the AI didn't return a variant for the detected OS, use the first one.
	if primary.Command == "" && len(variants) > 0 {
		primary = variants[0]
	}

	return CommandResponse{
		Primary:    primary,
		Variants:   variants,
		RiskLevel:  parsed.RiskLevel,
		RiskReason: parsed.RiskReason,
	}, nil
}

// stripMarkdownFences removes ``` or ```json fences that some models emit despite instructions.
func stripMarkdownFences(s string) string {
	// Remove leading fence.
	if strings.HasPrefix(s, "```") {
		idx := strings.Index(s, "\n")
		if idx != -1 {
			s = s[idx+1:]
		}
	}
	// Remove trailing fence.
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
