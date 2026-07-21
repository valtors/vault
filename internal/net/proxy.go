package net

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/valtors/vault/internal/store"
)

type Rule struct {
	Host   string `json:"host"`
	Action string `json:"action"`
}

type Policy struct {
	allowed []string
	blocked []string
	mu      sync.RWMutex
	logs    *store.DB
}

func NewPolicy(logs *store.DB) *Policy {
	return &Policy{
		allowed: []string{},
		blocked: []string{},
		logs:    logs,
	}
}

func (p *Policy) Allow(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, h := range p.allowed {
		if h == host {
			return
		}
	}
	p.allowed = append(p.allowed, host)
}

func (p *Policy) Block(host string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, h := range p.blocked {
		if h == host {
			return
		}
	}
	p.blocked = append(p.blocked, host)
}

func (p *Policy) Check(host string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	hostname := extractHost(host)

	for _, blocked := range p.blocked {
		if matchHost(hostname, blocked) {
			if p.logs != nil {
				p.logs.Log("net", "BLOCKED", fmt.Sprintf("connection to %s blocked by policy", host))
			}
			return fmt.Errorf("blocked: %s matches deny rule %s", host, blocked)
		}
	}

	if len(p.allowed) == 0 {
		return nil
	}

	for _, allowed := range p.allowed {
		if matchHost(hostname, allowed) {
			return nil
		}
	}

	if p.logs != nil {
		p.logs.Log("net", "BLOCKED", fmt.Sprintf("connection to %s blocked: not in allowlist", host))
	}
	return fmt.Errorf("blocked: %s not in allowlist", host)
}

func (p *Policy) Rules() []Rule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var rules []Rule
	for _, h := range p.allowed {
		rules = append(rules, Rule{Host: h, Action: "allow"})
	}
	for _, h := range p.blocked {
		rules = append(rules, Rule{Host: h, Action: "block"})
	}
	return rules
}

func (p *Policy) SetRules(rules []Rule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowed = p.allowed[:0]
	p.blocked = p.blocked[:0]
	for _, r := range rules {
		if r.Action == "allow" {
			p.allowed = append(p.allowed, r.Host)
		} else {
			p.blocked = append(p.blocked, r.Host)
		}
	}
}

type Dialer struct {
	policy *Policy
	timeout time.Duration
}

func NewDialer(policy *Policy, timeout time.Duration) *Dialer {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Dialer{policy: policy, timeout: timeout}
}

func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	if err := d.policy.Check(addr); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func matchHost(host, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == host {
		return true
	}
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		if len(host) >= len(suffix) && host[len(host)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

type ProxyStats struct {
	Allowed int `json:"allowed"`
	Blocked int `json:"blocked"`
}

func (p *Policy) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Rules())
}
