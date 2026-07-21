package env

import (
	"os"
	"reflect"
	"testing"
)

func TestSanitizeStripsTokens(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"GITHUB_TOKEN=ghp_secret123",
		"OPENAI_API_KEY=sk-abc",
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI",
		"HOME=/home/user",
		"TERM=xterm",
	}
	result := Sanitize(input, "/home/sandbox")
	for _, entry := range result {
		key, val, _ := splitEnv(entry)
		if isSensitive(key) {
			t.Fatalf("sensitive var %s=%s survived", key, val)
		}
	}
}

func TestSanitizeSetsHome(t *testing.T) {
	input := []string{"PATH=/usr/bin", "HOME=/home/original"}
	result := Sanitize(input, "/home/sandbox")
	home := ""
	for _, entry := range result {
		key, val, _ := splitEnv(entry)
		if key == "HOME" {
			home = val
		}
	}
	if home != "/home/sandbox" {
		t.Fatalf("HOME = %s, want /home/sandbox", home)
	}
}

func TestSanitizeAddsDefaults(t *testing.T) {
	input := []string{"PATH=/usr/bin"}
	result := Sanitize(input, "/home/sandbox")
	keys := make(map[string]bool)
	for _, entry := range result {
		key, _, _ := splitEnv(entry)
		keys[key] = true
	}
	for _, required := range []string{"HOME", "TERM", "LANG", "SHELL"} {
		if !keys[required] {
			t.Fatalf("missing default var: %s", required)
		}
	}
}

func TestSanitizeDeduplicates(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"PATH=/usr/local/bin",
		"HOME=/home/sandbox",
	}
	result := Sanitize(input, "/home/sandbox")
	count := 0
	for _, entry := range result {
		key, _, _ := splitEnv(entry)
		if key == "PATH" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("PATH appears %d times, want 1", count)
	}
}

func TestSanitizeDropsUnknownVars(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"MY_RANDOM_VAR=hello",
		"HOME=/home/sandbox",
	}
	result := Sanitize(input, "/home/sandbox")
	for _, entry := range result {
		key, _, _ := splitEnv(entry)
		if key == "MY_RANDOM_VAR" {
			t.Fatal("unknown var MY_RANDOM_VAR survived")
		}
	}
}

func TestSanitizePreservesPath(t *testing.T) {
	input := []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/home/sandbox"}
	result := Sanitize(input, "/home/sandbox")
	found := ""
	for _, entry := range result {
		key, val, _ := splitEnv(entry)
		if key == "PATH" {
			found = val
		}
	}
	if found != "/usr/local/bin:/usr/bin:/bin" {
		t.Fatalf("PATH = %s, want /usr/local/bin:/usr/bin:/bin", found)
	}
}

func TestIsSensitive(t *testing.T) {
	sensitive := []string{
		"GITHUB_TOKEN", "OPENAI_API_KEY", "AWS_SECRET_ACCESS_KEY",
		"ANTHROPIC_API_KEY", "CLAUDE_API_KEY", "DATABASE_URL",
		"NPM_TOKEN", "STRIPE_SECRET_KEY", "PRIVATE_KEY",
	}
	for _, key := range sensitive {
		if !IsSensitive(key) {
			t.Fatalf("%s should be sensitive", key)
		}
	}

	notSensitive := []string{"PATH", "HOME", "TERM", "LANG", "USER"}
	for _, key := range notSensitive {
		if IsSensitive(key) {
			t.Fatalf("%s should not be sensitive", key)
		}
	}
}

func TestSanitizeOS(t *testing.T) {
	os.Setenv("VAULT_TEST_TOKEN", "should-be-stripped")
	defer os.Unsetenv("VAULT_TEST_TOKEN")
	result := SanitizeOS("/home/sandbox")
	for _, entry := range result {
		key, _, _ := splitEnv(entry)
		if key == "VAULT_TEST_TOKEN" {
			t.Fatal("VAULT_TEST_TOKEN survived SanitizeOS")
		}
	}
}

func splitEnv(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return entry, "", false
}

func TestSplitEnv(t *testing.T) {
	k, v, ok := splitEnv("FOO=bar")
	if !ok || k != "FOO" || v != "bar" {
		t.Fatalf("splitEnv(FOO=bar) = (%s, %s, %v), want (FOO, bar, true)", k, v, ok)
	}
	_, _, ok = splitEnv("NOEQUALS")
	if ok {
		t.Fatal("splitEnv should return false for entry without =")
	}
}

func TestSanitizeEmptyInput(t *testing.T) {
	result := Sanitize([]string{}, "/home/sandbox")
	keys := make(map[string]bool)
	for _, entry := range result {
		key, _, _ := splitEnv(entry)
		keys[key] = true
	}
	if !keys["HOME"] {
		t.Fatal("empty input should still produce HOME")
	}
}

func TestSanitizeNilInput(t *testing.T) {
	result := Sanitize(nil, "/home/sandbox")
	if len(result) == 0 {
		t.Fatal("nil input should produce defaults")
	}
}

func TestSanitizeReflectsNoSecrets(t *testing.T) {
	input := []string{
		"PATH=/bin",
		"AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
		"AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"GITHUB_TOKEN=ghp_1234567890",
		"OPENAI_API_KEY=sk-proj-1234567890",
		"HOME=/home/sandbox",
	}
	result := SananitizeAssertNoSecrets(t, input, "/home/sandbox")
	_ = reflect.TypeOf(result)
}

func SananitizeAssertNoSecrets(t *testing.T, input []string, home string) []string {
	t.Helper()
	result := Sanitize(input, home)
	for _, entry := range result {
		if containsAny(entry, []string{"ghp_", "sk-", "AKIA", "wJalrXUtnFEMI"}) {
			t.Fatalf("secret value leaked into output: %s", entry)
		}
	}
	return result
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
		for i := 0; i <= len(s)-len(p); i++ {
			if s[i:i+len(p)] == p {
				return true
			}
		}
	}
	return false
}
