# Architecture

## Overview

Vault is a sandbox runtime for AI agents. It wraps untrusted processes in a filesystem overlay, sanitized environment, network policy, and prompt-injection scanner -- then logs every action to a per-sandbox audit database. The agent runs inside the sandbox; Vault watches, filters, and records.

```
                    +-------------+
                    | vault CLI   |
                    | (run/serve) |
                    +-----+-------+
                          |
              +-----------+-----------+
              |                       |
        +-----v-----+           +-----v-----+
        | sandbox    |          | api server |
        | (process)  |          | (HTTP)     |
        +-----+------+          +-----+------+
              |                       |
   +----------+----------+    +-------v-------+
   |          |          |    | sandbox map   |
   |     +----v---+  +---v--+  | (create/kill) |
   |     | fs     |  | env  |  +---------------+
   |     | overlay|  | sanitize
   |     +--------+  +------+
   |     +----v---+  +------v--+  +----------+
   |     | net    |  | mcp gate |  | audit db |
   |     | policy |  | (inject  |  | (sqlite) |
   |     +--------+  |  scan)   |  +----------+
   |                 +----------+
   v
  child process
  (the agent)
```

## Design Principles

1. **Sandbox the agent, not the user.** Vault wraps the agent process. The user starts Vault; Vault starts the agent inside a restricted environment. The agent never sees the real filesystem or real environment.
2. **Defense in depth.** Filesystem overlay blocks sensitive paths. Environment sanitizer strips secrets. Network policy filters connections. MCP gate scans tool descriptions for prompt injection. Each layer is independent; failure of one does not compromise the others.
3. **Audit everything.** Every sandbox lifecycle event, network connection attempt, and MCP tool call is logged to a per-sandbox SQLite database. Queryable after the fact.
4. **No dependencies on container primitives.** No namespaces, no cgroups, no seccomp. Pure Go process wrapping. Portable to any OS Go runs on.

## Components

### sandbox (`internal/sandbox`)

The core orchestrator. Creates the overlay, opens the audit DB, spawns the child process with sanitized env and restricted filesystem, and waits for completion.

**Lifecycle:**
1. `New(cfg)` -- assign atomic ID, create root dir (0700), open audit DB, create overlay.
2. `Start()` -- build `exec.Cmd` with sanitized env, overlay home as working dir, stdin/stdout/stderr passthrough. Optional timeout via `context.WithTimeout`.
3. `Wait()` -- block on child process completion. Log duration and exit status.
4. `Kill()` -- send SIGKILL to child process. Log the kill.
5. `Cleanup()` -- close DB, remove overlay filesystem.

**Config:**
| Field | Default | Purpose |
|---|---|---|
| `RootDir` | tempdir/vault-sandbox-N | Sandbox root directory |
| `AllowedDirs` | none | Directories symlinked into the sandbox |
| `AllowedHosts` | none (allow all) | Network allowlist |
| `BlockedHosts` | none | Network blocklist |
| `MaxMemoryMB` | 512 | Memory limit (advisory) |
| `MaxCPUSeconds` | 300 | CPU limit (advisory) |
| `TimeoutSecs` | 0 (unlimited) | Process timeout |
| `Command` | required | Command to run |
| `Args` | none | Command arguments |

### fs (`internal/fs`)

Filesystem overlay. Creates an isolated home directory and tmp directory under the sandbox root. Symlinks allowed directories into a controlled `allowed/` subdirectory.

**Path resolution:** `Resolve(path)` checks:
1. Path is not in a blocked list (`.ssh`, `.aws`, `.gnupg`, `.docker`, `.kube`, `.npmrc`, `.pypirc`, `.netrc`, `.env`, `.gitconfig`, etc.).
2. Path is inside sandbox home, sandbox tmp, or the allowed symlinks directory.
3. Otherwise: blocked.

**Blocked paths:** 12 sensitive dotfiles/directories are hard-blocked. The agent cannot read SSH keys, cloud credentials, API tokens, or git config even if it escapes the overlay.

### env (`internal/env`)

Environment sanitizer. Strips all variables except a safe whitelist, then filters out anything matching sensitive patterns.

**Two-stage filtering:**
1. **Whitelist:** Only `PATH`, `TERM`, `LANG`, `LC_ALL`, `LC_CTYPE`, `HOME`, `SHELL`, `USER`, `LOGNAME`, `TMPDIR`, `TMP`, `TEMP`, `SHLVL`, `PWD`, `OLDPWD`, `_`, `XDG_RUNTIME_DIR` pass through.
2. **Pattern filter:** Any env var whose name matches sensitive regex patterns (token, secret, password, credential, api_key, auth, aws_, azure_, google, openai, anthropic, stripe, etc.) is stripped.

**HOME rewriting:** `HOME` is set to the overlay home directory, not the real home. The agent thinks it is in its own home.

**Defaults:** `TERM=xterm-256color`, `LANG=C.UTF-8`, `SHELL=/bin/sh` are set if missing.

### net (`internal/net`)

Network policy engine. Allowlist/blocklist with wildcard support.

**Policy evaluation:**
1. Check blocklist first. If host matches any blocked pattern, deny.
2. If allowlist is empty, allow (default open).
3. If allowlist is non-empty, host must match an allow rule.

**Wildcard matching:** `*.example.com` matches any subdomain. `*` matches everything.

**Dialer:** `Dial(network, addr)` checks policy before connecting. Blocked connections are logged to the audit DB. Timeout configurable (default 10s).

### mcp gate (`internal/mcp`)

MCP proxy with prompt-injection scanning. Sits between the MCP client and the target MCP server, intercepts `tools/list` responses, scans each tool description for injection patterns, and strips them before forwarding.

**Injection scanner** (`internal/inject`):
- 30 regex patterns across 10 categories: `prompt_override`, `identity_swap`, `exfiltration`, `destructive`, `pipe_to_shell`, `base64_obfuscation`, `data_theft`, `privilege_escalation`, `network_scan`, `reverse_shell`, `tool_poisoning`.
- Severity levels: CRITICAL (25 pts), HIGH (15 pts), MEDIUM (8 pts), default (3 pts). Capped at 100.
- `Scan(text, tool)` -- returns findings. `Strip(text)` -- removes matched text, replaces with `[stripped: <pattern>]`. `ScanDescription` -- returns a `Result` with clean/dirty flag.

**Proxy flow:**
1. Forward client stdin to server stdin.
2. Read server stdout line by line (JSON-RPC over newline-delimited JSON).
3. If `tools/list` response: scan each tool description, strip injections, rewrite response.
4. If `tools/call`: log tool name to audit DB.
5. Forward all messages to client.

**ScanTools:** Standalone mode. Sends `initialize` + `tools/list` to a server, scans all tool descriptions, returns results without proxying.

### store (`internal/store`)

Audit log. SQLite with a single `audit` table.

```sql
CREATE TABLE audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    category TEXT NOT NULL,
    action TEXT NOT NULL,
    detail TEXT
);
```

Categories: `sandbox` (lifecycle), `net` (connection attempts), `mcp` (tool calls, injection findings).

WAL mode, 5-second busy timeout. Mutex-protected for concurrent access.

### api (`internal/api`)

HTTP server for managing sandboxes remotely.

| Endpoint | Method | Purpose |
|---|---|---|
| `/health` | GET | Health check |
| `/sandboxes` | POST | Create and start a sandbox |
| `/sandboxes/{id}` | GET | Sandbox status |
| `/sandboxes/{id}` | DELETE | Kill and cleanup sandbox |
| `/sandboxes/{id}/audit` | GET | Query audit log |

Server tracks sandboxes in a `map[int64]*sandbox.Sandbox` protected by `sync.Mutex`.

## Process Model

Two modes:
- **`vault run`** -- start a single sandbox, wait for it to finish, exit. Signal forwarding (SIGINT/SIGTERM kills the child).
- **`vault serve`** -- start the HTTP API server, manage multiple sandboxes concurrently.

## File Layout

```
<sandbox-root>/
  audit.db          # SQLite audit log (WAL mode)
  audit.db-wal
  audit.db-shm
  home/             # Agent's HOME directory
  tmp/              # Agent's TMPDIR
  allowed/          # Symlinks to allowed directories
```

## Testing

83 tests, 76.3% coverage. Race detector enabled in CI. Tests cover:
- Sandbox lifecycle: create, start, wait, kill, cleanup
- Overlay: path resolution, blocked paths, allowed symlinks, cleanup
- Env sanitizer: sensitive pattern detection, whitelist enforcement, HOME rewrite, defaults
- Network policy: allow/block/deny, wildcard matching, dialer
- Injection scanner: all 30 patterns, strip, risk score, clean/dirty detection
- MCP gate: tool list interception, injection stripping, scan mode
- Store: log, query, count, concurrent access
- API: health, create, status, kill, audit query

## Dependencies

- `modernc.org/sqlite` -- Pure-Go SQLite (no CGO needed)
- Go stdlib for everything else
