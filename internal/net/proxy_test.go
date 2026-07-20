package net

import (
	"testing"
)

func TestCheckEmptyAllowlist(t *testing.T) {
	p := NewProxy(nil, nil)
	if err := p.Check("example.com"); err != nil {
		t.Fatalf("empty allowlist should allow all: %v", err)
	}
}

func TestCheckBlocked(t *testing.T) {
	p := NewProxy(nil, []string{"evil.com"})
	if err := p.Check("evil.com"); err == nil {
		t.Fatal("blocked host should be denied")
	}
}

func TestCheckAllowedOnly(t *testing.T) {
	p := NewProxy([]string{"good.com"}, nil)
	if err := p.Check("good.com"); err != nil {
		t.Fatalf("allowed host should pass: %v", err)
	}
	if err := p.Check("bad.com"); err == nil {
		t.Fatal("non-allowlisted host should be denied")
	}
}

func TestCheckWildcardBlocked(t *testing.T) {
	p := NewProxy(nil, []string{"*.evil.com"})
	if err := p.Check("sub.evil.com"); err == nil {
		t.Fatal("wildcard block should match subdomain")
	}
}

func TestCheckPortStripped(t *testing.T) {
	p := NewProxy(nil, []string{"evil.com"})
	if err := p.Check("evil.com:443"); err == nil {
		t.Fatal("port should be stripped before matching")
	}
}

func TestDynamicBlock(t *testing.T) {
	p := NewProxy(nil, nil)
	p.Block("newbad.com")
	if err := p.Check("newbad.com"); err == nil {
		t.Fatal("dynamically blocked host should be denied")
	}
}

func TestDynamicAllow(t *testing.T) {
	p := NewProxy([]string{"onlythis.com"}, nil)
	p.Allow("alsothis.com")
	if err := p.Check("alsothis.com"); err != nil {
		t.Fatal("dynamically allowed host should pass")
	}
}

func TestRules(t *testing.T) {
	p := NewProxy([]string{"good.com"}, []string{"bad.com"})
	rules := p.Rules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}
