# vault

[![Go Report Card](https://goreportcard.com/badge/github.com/valtors/vault?style=flat-square)](https://goreportcard.com/report/github.com/valtors/vault)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?style=flat-square)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-MIT-0F172A?style=flat-square)](LICENSE)
[![CI](https://github.com/valtors/vault/actions/workflows/ci.yml/badge.svg)](https://github.com/valtors/vault/actions/workflows/ci.yml)
[![tests](https://img.shields.io/badge/tests-75-green?style=flat-square)]()

run your agent. it can't destroy your machine.

## what

vault is a sandbox for ai agents. you run a command inside it. the agent thinks it has full access to your system. it doesn't.

[landing](https://valtors.github.io/vault/) - [github](https://github.com/valtors/vault)

- **filesystem overlay** - agent gets a fake home directory. `~/.ssh` is invisible. `~/.aws` is invisible. `~/.env` is invisible. writes go to the overlay. reads from allowlisted paths only.
- **env sanitizer** - strips every secret from the environment. tokens, api keys, credentials, passwords. gone. the agent sees a clean shell.
- **network policy** - allow/deny rules per host. wildcard support. the agent can't call your production database. the agent can't exfiltrate data. every connection logged.
- **mcp gate** - every mcp server connection goes through the scanner. tool descriptions are checked for prompt injection. injection patterns are stripped before the agent sees them.
- **inject scanner** - 30 patterns covering prompt override, identity swap, exfiltration, destructive commands, reverse shells, tool poisoning, base64 obfuscation, privilege escalation.
- **audit log** - sqlite. every sandbox action, every file access, every network request, every injection attempt. timestamped. queryable.
- **http api** - create sandboxes, query audit logs, kill processes, manage rules. all from a single endpoint.

## why not just X

| | docker | firejail | vault |
|---|---|---|---|
| filesystem isolation | container fs | mount namespace | overlay fs |
| env sanitization | no | no | yes |
| mcp injection scanning | no | no | yes |
| audit log | volume logs | no | sqlite, queryable |
| setup time | minutes | minutes | zero config |
| runtime overhead | high | medium | low |
| agent-aware | no | no | yes |

docker isolates everything but doesn't know about agents. firejail isolates processes but doesn't scan for prompt injection. vault does both: isolates the system and understands what an agent is.

## install

```bash
go install github.com/valtors/vault/cmd/vault@latest
```

## use

run a command in a sandbox:

```bash
vault run -- claude-code
vault run -timeout 60 -- python script.py
vault run -allow /home/user/project -- npm test
```

start the api server:

```bash
vault serve -port 9090
```

api:

```bash
curl -X POST localhost:9090/sandboxes -d '{"command":"echo","args":["test"]}'
curl localhost:9090/sandboxes
curl localhost:9090/sandboxes/1/logs
curl -X POST localhost:9090/sandboxes/1/kill
```

## how it works

```
+----------------------------------------------+
|  vault                                        |
|                                               |
|  +----------+  +----------+  +--------------+ |
|  | env       |  | fs       |  | net          | |
|  | sanitizer |  | overlay  |  | policy       | |
|  +----------+  +----------+  +--------------+ |
|                                               |
|  +----------+  +----------+  +--------------+ |
|  | mcp gate |  | inject   |  | audit log    | |
|  |          |  | scanner  |  | (sqlite)     | |
|  +----------+  +----------+  +--------------+ |
|                                               |
|  +-----------------------------------------+  |
|  |  http api (create/kill/logs/rules)      |  |
|  +-----------------------------------------+  |
+----------------------------------------------+
                    |
              +-----+-----+
              |  agent     |  thinks it has root. doesn't.
              +-----------+
```

## what gets stripped

env vars matching: token, secret, password, credential, api_key, auth, aws_, azure_, google, openai, anthropic, claude, stripe, resend, mailgun, sendgrid, database_url, dsn, private_key, ssh, npm_token, github_token, gh_pat, and anything else that looks like a secret.

blocked paths: `.ssh`, `.aws`, `.gnupg`, `.docker`, `.kube`, `.config/gcloud`, `.config/gh`, `.npmrc`, `.pypirc`, `.netrc`, `.env`, `.gitconfig`.

injection patterns: prompt override, identity swap, exfiltration, destructive commands, reverse shells, tool poisoning, base64 obfuscation, privilege escalation, network scanning, data theft, pipe-to-shell. 30 patterns total. all stripped before the agent sees them.

## tests

75 tests. 72.5% coverage. all pass.

```bash
go test ./internal/... -race
```

## tech

go. single binary. zero runtime dependencies. sqlite (pure-go, no cgo). stdlib everything else. boring tech on purpose.

## license

MIT
