package inject

import (
	"regexp"
	"strings"
)

type Finding struct {
	Tool       string `json:"tool"`
	Pattern    string `json:"pattern"`
	Severity   string `json:"severity"`
	Snippet    string `json:"snippet"`
}

var patterns = []struct {
	regex    *regexp.Regexp
	name     string
	severity string
}{
	{regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous\s+)?instructions`), "prompt_override", "critical"},
	{regexp.MustCompile(`(?i)disregard\s+(the\s+)?(above|previous|prior)`), "prompt_override", "critical"},
	{regexp.MustCompile(`(?i)forget\s+(everything|all\s+(prior|previous))`), "prompt_override", "critical"},
	{regexp.MustCompile(`(?i)you\s+are\s+(now|actually)\s+`), "identity_swap", "high"},
	{regexp.MustCompile(`(?i)new\s+instructions?\s*:`), "instruction_injection", "high"},
	{regexp.MustCompile(`(?i)do\s+not\s+(follow|obey)\s+(your|the)\s+(rules|guidelines)`), "rule_bypass", "high"},
	{regexp.MustCompile(`(?i)system\s*:\s*`), "system_prefix", "medium"},
	{regexp.MustCompile(`(?i)<\s*system\s*>`), "system_tag", "medium"},
	{regexp.MustCompile(`(?i)exfiltrat|upload\s+(to|all)|send\s+(to|all)\s+(remote|external)`), "exfiltration", "high"},
	{regexp.MustCompile(`(?i)rm\s+-rf\s+/`), "destructive_cmd", "critical"},
	{regexp.MustCompile(`(?i)(curl|wget)\s+.*\|\s*(sh|bash)`), "pipe_to_shell", "critical"},
	{regexp.MustCompile(`(?i)base64\s+-d\s*\|`), "b64_pipe", "high"},
	{regexp.MustCompile(`(?i)eval\s*\(`), "eval_call", "medium"},
	{regexp.MustCompile(`(?i)exec\s*\(`), "exec_call", "medium"},
}

func Scan(text string, toolName string) []Finding {
	var findings []Finding
	lower := strings.ToLower(text)

	for _, p := range patterns {
		if p.regex.MatchString(text) {
			loc := p.regex.FindStringIndex(text)
			snippet := text
			if loc != nil {
				start := loc[0]
				end := loc[1]
				if start > 50 {
					start -= 50
				}
				if end+50 < len(text) {
					end += 50
				}
				snippet = text[start:end]
			}

			findings = append(findings, Finding{
				Tool:     toolName,
				Pattern:  p.name,
				Severity: p.severity,
				Snippet:  snippet,
			})
		}
	}

	_ = lower
	return findings
}

func Strip(text string) string {
	stripped := text
	for _, p := range patterns {
		if p.severity == "critical" || p.severity == "high" {
			stripped = p.regex.ReplaceAllString(stripped, "[blocked]")
		}
	}
	return stripped
}

func HasInjection(text string) bool {
	return len(Scan(text, "")) > 0
}
