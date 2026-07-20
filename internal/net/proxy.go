package net

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type Rule struct {
	Host   string `json:"host"`
	Action string `json:"action"`
}

type Proxy struct {
	allowed []string
	blocked []string
	mu      sync.RWMutex
}

func NewProxy(allowed, blocked []string) *Proxy {
	return &Proxy{
		allowed: allowed,
		blocked: blocked,
	}
}

func (p *Proxy) Allow(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowed = append(p.allowed, host)
}

func (p *Proxy) Block(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked = append(p.blocked, host)
}

func (p *Proxy) Check(host string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	hostname := strings.ToLower(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = strings.ToLower(h)
	}

	for _, b := range p.blocked {
		if matchHost(hostname, strings.ToLower(b)) {
			return fmt.Errorf("blocked: %s", host)
		}
	}

	if len(p.allowed) == 0 {
		return nil
	}

	for _, a := range p.allowed {
		if matchHost(hostname, strings.ToLower(a)) {
			return nil
		}
	}

	return fmt.Errorf("not in allowlist: %s", host)
}

func (p *Proxy) Rules() []Rule {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var rules []Rule
	for _, h := range p.blocked {
		rules = append(rules, Rule{Host: h, Action: "deny"})
	}
	for _, h := range p.allowed {
		rules = append(rules, Rule{Host: h, Action: "allow"})
	}
	return rules
}

func matchHost(host, pattern string) bool {
	if host == pattern {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return strings.HasSuffix(host, suffix)
	}
	return false
}
