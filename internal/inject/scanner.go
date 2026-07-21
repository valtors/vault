package inject

import (
	"regexp"
)

type Finding struct {
	Tool     string `json:"tool"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
	Snippet  string `json:"snippet"`
}

type Result struct {
	Tool     string    `json:"tool"`
	Clean    bool      `json:"clean"`
	Findings []Finding `json:"findings"`
}

var patterns = []struct {
	name     string
	severity string
	regex    *regexp.Regexp
}{
	{"prompt_override", "CRITICAL", regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above|earlier)\s+instructions?`)},
	{"prompt_override", "CRITICAL", regexp.MustCompile(`(?i)disregard\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?)`)},
	{"prompt_override", "CRITICAL", regexp.MustCompile(`(?i)forget\s+(everything|all|previous)\s+(you|that|which)\s+(know|read|were\s+told)`)},
	{"prompt_override", "CRITICAL", regexp.MustCompile(`(?i)you\s+are\s+now\s+(a|an)\s+\w+`)},
	{"identity_swap", "CRITICAL", regexp.MustCompile(`(?i)you\s+are\s+(not|no\s+longer)\s+(an?\s+)?(ai|assistant|agent|model)`)},
	{"identity_swap", "HIGH", regexp.MustCompile(`(?i)pretend\s+(you\s+are|to\s+be)\s+(a|an)\s+\w+`)},
	{"identity_swap", "HIGH", regexp.MustCompile(`(?i)act\s+as\s+(if\s+you\s+are\s+)?(a|an)\s+(different|new)\s+\w+`)},
	{"exfiltration", "CRITICAL", regexp.MustCompile(`(?i)(send|upload|post|exfiltrate|transmit|transfer)\s+(the\s+)?(file\s+)?(data|contents?|secrets?|tokens?|keys?|passwords?|credentials?)\s+to\s+(https?://|ftp://|http://)`)},
	{"exfiltration", "CRITICAL", regexp.MustCompile(`(?i)(send|upload|post)\s+(to|via)\s+(webhook|discord|telegram|slack|pastebin)`)},
	{"exfiltration", "HIGH", regexp.MustCompile(`(?i)(read|cat|type|print|show|display|fetch|get)\s+(the\s+)?(contents?\s+of\s+)?(~/?|/home/|/etc/|/\.|\$HOME/)?\.?(ssh|aws|gnupg|env|bashrc|bash_history|zshrc|netrc|npmrc|pypirc|gitconfig)`)},
	{"destructive", "CRITICAL", regexp.MustCompile(`(?i)(rm\s+-rf|del\s+/[sS]|format\s+[cC]:|rmdir\s+/|wipe\s+disk|dd\s+if=/dev/zero)`)},
	{"destructive", "HIGH", regexp.MustCompile(`(?i)(drop\s+table|drop\s+database|truncate\s+table|delete\s+from\s+\w+\s+where\s+1=1)`)},
	{"destructive", "HIGH", regexp.MustCompile(`(?i)(kill\s+-9\s+1|killall|pkill|shutdown|reboot|halt\s+system)`)},
	{"pipe_to_shell", "HIGH", regexp.MustCompile(`(?i)\|\s*(sh|bash|zsh|ksh|dash|/bin/sh|/bin/bash)`)},
	{"pipe_to_shell", "HIGH", regexp.MustCompile(`(?i)(exec|eval|system|popen|subprocess)\s*\(\s*["'].*["']\s*\)`)},
	{"base64_obfuscation", "MEDIUM", regexp.MustCompile(`(?i)(base64|btoa|atob)\s*[\(\[].{10,}[\)\]]`)},
	{"base64_obfuscation", "HIGH", regexp.MustCompile(`(?i)(echo|printf)\s+["']?[A-Za-z0-9+/]{40,}={0,2}["']?\s*\|\s*(base64\s+-d|sh|bash)`)},
	{"data_theft", "HIGH", regexp.MustCompile(`(?i)(copy|cp|scp|rsync)\s+(~/|/home/|/root/|/etc/)`)},
	{"data_theft", "HIGH", regexp.MustCompile(`(?i)(curl|wget|fetch|http)\s+.*\s+--data\s+.*(?:token|key|secret|password|credential)`)},
	{"privilege_escalation", "HIGH", regexp.MustCompile(`(?i)(sudo\s+|su\s+|chmod\s+[47]77|chown\s+root)`)},
	{"network_scan", "MEDIUM", regexp.MustCompile(`(?i)(nmap|masscan|zmap|netcat|nc\s+-z)\s`)},
	{"reverse_shell", "CRITICAL", regexp.MustCompile(`(?i)(bash|sh|zsh)\s+-i\s+>\s*&\s*/dev/tcp/`)},
	{"reverse_shell", "CRITICAL", regexp.MustCompile(`(?i)(nc\s+-l\s+-p|netcat\s+-l\s+-p|socat\s+TCP-LISTEN)`)},
	{"tool_poisoning", "CRITICAL", regexp.MustCompile(`(?i)when\s+(called|invoked|executed|run),?\s+(instead|first|also|always)\s+(run|execute|call|invoke|send)`)},
	{"tool_poisoning", "CRITICAL", regexp.MustCompile(`(?i)before\s+(running|executing|returning|responding),?\s+(first\s+)?(run|execute|call|send|fetch)\s+(this|the\s+following|a\s+command)`)},
	{"tool_poisoning", "HIGH", regexp.MustCompile(`(?i)secretly\s+(run|execute|call|send|fetch|upload)`)},
	{"tool_poisoning", "HIGH", regexp.MustCompile(`(?i)without\s+(the\s+user|telling\s+the\s+user|user'?s?\s+knowledge|user'?s?\s+consent)`)},
	{"tool_poisoning", "MEDIUM", regexp.MustCompile(`(?i)also\s+(run|execute|call|send|fetch|read)\s+(the\s+following|this\s+command|this\s+url)`)},
}

func Scan(text, tool string) []Finding {
	var findings []Finding
	for _, p := range patterns {
		matches := p.regex.FindAllString(text, -1)
		for _, match := range matches {
			findings = append(findings, Finding{
				Tool:     tool,
				Pattern:  p.name,
				Severity: p.severity,
				Snippet:  truncate(match, 200),
			})
		}
	}
	return findings
}

func ScanDescription(desc, tool string) Result {
	findings := Scan(desc, tool)
	return Result{
		Tool:     tool,
		Clean:    len(findings) == 0,
		Findings: findings,
	}
}

func Strip(text string) (string, []Finding) {
	var stripped []Finding
	result := text
	for _, p := range patterns {
		matches := p.regex.FindAllStringIndex(result, -1)
		if len(matches) == 0 {
			continue
		}
		for i := len(matches) - 1; i >= 0; i-- {
			start, end := matches[i][0], matches[i][1]
			match := result[start:end]
			stripped = append(stripped, Finding{
				Pattern:  p.name,
				Severity: p.severity,
				Snippet:  truncate(match, 200),
			})
			result = result[:start] + "[stripped: " + p.name + "]" + result[end:]
		}
	}
	return result, stripped
}

func RiskScore(findings []Finding) int {
	if len(findings) == 0 {
		return 0
	}
	score := 0
	for _, f := range findings {
		switch f.Severity {
		case "CRITICAL":
			score += 25
		case "HIGH":
			score += 15
		case "MEDIUM":
			score += 8
		default:
			score += 3
		}
	}
	if score > 100 {
		score = 100
	}
	return score
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
