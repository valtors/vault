package net

import (
	"testing"
)

func TestCheckEmptyAllowlistAllowsAll(t *testing.T) {
	p := NewPolicy(nil)
	if err := p.Check("example.com:443"); err != nil {
		t.Fatalf("empty allowlist should allow: %v", err)
	}
}

func TestCheckAllowlistPermits(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("api.openai.com")
	if err := p.Check("api.openai.com:443"); err != nil {
		t.Fatalf("allowlisted host blocked: %v", err)
	}
}

func TestCheckAllowlistBlocks(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("api.openai.com")
	if err := p.Check("evil.com:443"); err == nil {
		t.Fatal("non-allowlisted host should be blocked")
	}
}

func TestCheckBlocklistBlocks(t *testing.T) {
	p := NewPolicy(nil)
	p.Block("evil.com")
	if err := p.Check("evil.com:443"); err == nil {
		t.Fatal("blocked host should be blocked")
	}
}

func TestCheckBlocklistAllowsOthers(t *testing.T) {
	p := NewPolicy(nil)
	p.Block("evil.com")
	if err := p.Check("good.com:443"); err != nil {
		t.Fatalf("non-blocked host should pass: %v", err)
	}
}

func TestMatchHostExact(t *testing.T) {
	if !matchHost("example.com", "example.com") {
		t.Fatal("exact match should work")
	}
}

func TestMatchHostWildcard(t *testing.T) {
	if !matchHost("api.example.com", "*.example.com") {
		t.Fatal("wildcard suffix should match")
	}
}

func TestMatchHostWildcardAll(t *testing.T) {
	if !matchHost("anything.com", "*") {
		t.Fatal("* should match everything")
	}
}

func TestMatchHostNoMatch(t *testing.T) {
	if matchHost("example.com", "evil.com") {
		t.Fatal("non-matching hosts should not match")
	}
}

func TestMatchHostPartialWildcard(t *testing.T) {
	if matchHost("notexample.com", "*.example.com") {
		t.Fatal("partial wildcard should not match")
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com:443", "example.com"},
		{"192.168.1.1:8080", "192.168.1.1"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		got := extractHost(tt.input)
		if got != tt.want {
			t.Fatalf("extractHost(%s) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestSetRules(t *testing.T) {
	p := NewPolicy(nil)
	p.SetRules([]Rule{
		{Host: "api.openai.com", Action: "allow"},
		{Host: "evil.com", Action: "block"},
	})
	if err := p.Check("api.openai.com:443"); err != nil {
		t.Fatalf("allow rule not applied: %v", err)
	}
	if err := p.Check("evil.com:443"); err == nil {
		t.Fatal("block rule not applied")
	}
}

func TestSetRulesReplaces(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("old.com")
	p.SetRules([]Rule{{Host: "new.com", Action: "allow"}})
	if err := p.Check("old.com:443"); err == nil {
		t.Fatal("old rule should be replaced")
	}
	if err := p.Check("new.com:443"); err != nil {
		t.Fatalf("new rule not applied: %v", err)
	}
}

func TestRules(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("a.com")
	p.Block("b.com")
	rules := p.Rules()
	if len(rules) != 2 {
		t.Fatalf("Rules() returned %d, want 2", len(rules))
	}
}

func TestAllowDedup(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("a.com")
	p.Allow("a.com")
	if len(p.allowed) != 1 {
		t.Fatalf("duplicate allow should dedup, got %d", len(p.allowed))
	}
}

func TestBlockDedup(t *testing.T) {
	p := NewPolicy(nil)
	p.Block("b.com")
	p.Block("b.com")
	if len(p.blocked) != 1 {
		t.Fatalf("duplicate block should dedup, got %d", len(p.blocked))
	}
}

func TestBlockOverridesAllow(t *testing.T) {
	p := NewPolicy(nil)
	p.Allow("a.com")
	p.Block("a.com")
	if err := p.Check("a.com:443"); err == nil {
		t.Fatal("block should override allow")
	}
}
