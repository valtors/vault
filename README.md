# vault

[![Go Report Card](https://goreportcard.com/badge/github.com/valtors/vault?style=flat-square)](https://goreportcard.com/report/github.com/valtors/vault)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?style=flat-square)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

run your agent. it can't destroy your machine.

vault is a sandbox for AI agents. you run your agent inside it. the agent thinks it has full access to your filesystem, your network, your environment. it doesn't.

every file access goes through an overlay. every network connection goes through a proxy. every MCP server gets scanned for prompt injection before it connects. every tool call gets logged. every injection attempt gets blocked.

the agent can't read your ssh keys. it can't call your production database. it can't exfiltrate data. it can't follow instructions embedded in tool descriptions. and if it tries, you'll know.

## install

```
go install github.com/valtors/vault/cmd/vault@latest
```

or build from source:

```
git clone https://github.com/valtors/vault
cd vault
go build -o vault ./cmd/vault/
```

one binary. no daemon. no config file. no dependencies beyond the go standard library and a pure-go sqlite driver.

## usage

run a command inside the sandbox:

```
vault run -- claude-code
vault run -- npx -y @modelcontextprotocol/server-filesystem /tmp
vault run -- python agent.py
```

start the HTTP API:

```
vault serve -port 9090
```

query the audit log:

```
curl localhost:9090/audit?source=mcp_gate
curl localhost:9090/audit?event=injection_blocked
curl localhost:9090/audit?limit=50
```

## what it does

**filesystem overlay.** the agent gets a sandboxed home. it can't see `~/.ssh`, `~/.aws`, `~/.config`, or anything you block. allowed directories pass through read-only. writes go to the overlay, not your real filesystem.

**network proxy.** every outbound connection passes through an allow/deny ruleset. the agent can't call your production database. it can't exfiltrate to a random host. you set the rules. the agent follows them or gets denied.

**mcp gate.** every MCP server the agent tries to connect to gets probed first. tool descriptions are scanned for prompt injection. if a tool says "ignore all previous instructions" in its description, it gets stripped before the agent sees it. if a tool tries to pipe to shell or run destructive commands, it gets flagged. the leaderboard proved 11 out of 17 npm MCP servers had security findings. vault blocks them.

**injection detector.** scans tool descriptions and responses for 15 injection patterns. prompt overrides, identity swaps, system prefix injection, exfiltration attempts, destructive commands, pipe-to-shell, base64 pipe obfuscation, eval/exec calls. critical and high severity patterns get stripped. the rest get logged.

**audit log.** sqlite database. every tool call, every file access, every network request, every injection attempt. timestamped. queryable. "what did the agent do at 3am" is one curl.

## how it works

vault uses linux namespace isolation (`CLONE_NEWNS`, `CLONE_NEWPID`) to give the agent its own mount and pid namespace. the filesystem overlay redirects writes to a sandbox directory. the network proxy intercepts outbound connections and applies rules. the mcp gate speaks the MCP protocol directly, probing servers for tool lists before the agent ever connects.

the agent never knows it's sandboxed. it runs normally. it sees a filesystem. it makes network calls. it connects to MCP servers. everything works. until it tries something it shouldn't.

## architecture

```
cmd/vault/          entrypoint
internal/
  sandbox/          process isolation, namespace config
  fs/               overlay filesystem with allow/deny
  net/              network proxy with host rules
  mcp/              mcp server gate with injection scanning
  inject/           prompt injection pattern scanner
  store/             sqlite audit log
  api/               http api server
```

## why

every developer using AI agents is giving them root. claude, cursor, any agent. they get your filesystem, your network, your api keys. and you hope nothing goes wrong.

you connected an MCP server to your AI agent. you didn't check what was inside. some of these servers tell your agent to ignore your instructions. and your agent will.

vault is the answer. run your agent in it. the agent can't destroy your machine.

## license

MIT
