package env

import (
	"os"
	"regexp"
	"strings"
)

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)passwd`),
	regexp.MustCompile(`(?i)credential`),
	regexp.MustCompile(`(?i)api[_-]?key`),
	regexp.MustCompile(`(?i)auth`),
	regexp.MustCompile(`(?i)aws_`),
	regexp.MustCompile(`(?i)azure_`),
	regexp.MustCompile(`(?i)google`),
	regexp.MustCompile(`(?i)gcloud`),
	regexp.MustCompile(`(?i)database_url`),
	regexp.MustCompile(`(?i)dsn`),
	regexp.MustCompile(`(?i)private[_-]?key`),
	regexp.MustCompile(`(?i)ssh`),
	regexp.MustCompile(`(?i)npm_token`),
	regexp.MustCompile(`(?i)github_token`),
	regexp.MustCompile(`(?i)gh[_-]?pat`),
	regexp.MustCompile(`(?i)openai`),
	regexp.MustCompile(`(?i)anthropic`),
	regexp.MustCompile(`(?i)claude`),
	regexp.MustCompile(`(?i)resend`),
	regexp.MustCompile(`(?i)mailgun`),
	regexp.MustCompile(`(?i)sendgrid`),
	regexp.MustCompile(`(?i)stripe`),
	regexp.MustCompile(`(?i)twilio`),
}

var safeVars = map[string]bool{
	"PATH":            true,
	"TERM":            true,
	"LANG":            true,
	"LC_ALL":          true,
	"LC_CTYPE":        true,
	"HOME":            true,
	"SHELL":           true,
	"USER":            true,
	"LOGNAME":         true,
	"TMPDIR":          true,
	"TMP":             true,
	"TEMP":            true,
	"SHLVL":           true,
	"PWD":             true,
	"OLDPWD":          true,
	"_":               true,
	"XDG_RUNTIME_DIR": true,
}

var defaultSafeValues = map[string]string{
	"TERM": "xterm-256color",
	"LANG": "C.UTF-8",
	"SHELL": "/bin/sh",
}

func Sanitize(environ []string, home string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, entry := range environ {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]

		if seen[key] {
			continue
		}

		if isSensitive(key) {
			continue
		}

		if !isSafe(key) {
			continue
		}

		if key == "HOME" {
			val = home
		}

		seen[key] = true
		result = append(result, key+"="+val)
	}

	if !seen["HOME"] {
		result = append(result, "HOME="+home)
	}
	if !seen["TERM"] {
		result = append(result, "TERM="+defaultSafeValues["TERM"])
	}
	if !seen["LANG"] {
		result = append(result, "LANG="+defaultSafeValues["LANG"])
	}
	if !seen["SHELL"] {
		result = append(result, "SHELL="+defaultSafeValues["SHELL"])
	}

	return result
}

func SanitizeOS(home string) []string {
	return Sanitize(os.Environ(), home)
}

func isSensitive(key string) bool {
	upper := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(upper) {
			return true
		}
	}
	return false
}

func isSafe(key string) bool {
	return safeVars[key]
}

func IsSensitive(key string) bool {
	return isSensitive(key)
}
