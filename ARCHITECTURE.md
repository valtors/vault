# Architecture

## Overview

Vault is an experimental policy and audit runtime for AI agents. It launches a host process with an alternate working directory and sanitized environment, provides opt-in filesystem/network helpers, scans MCP metadata, and records selected actions. The current implementation is not an OS sandbox and must not be used to execute untrusted code.

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

1. **Do not overstate the boundary.** Vault does not currently isolate an untrusted child process from the host kernel, filesystem, or network.
2. **Reduce accidental exposure.** Environment sanitation, alternate working directories, policy helpers, and MCP metadata scanning are defense-supporting controls, not containment.
3. **Make recorded behavior inspectable.** Selected lifecycle events and operations routed through Vault can be written to a per-process SQLite database.
4. **Require real isolation for untrusted execution.** A production security boundary must use and verify OS/container/VM primitives such as namespaces plus seccomp, Landlock, gVisor, Kata, or Firecracker.

## Components

### sandbox (`internal/sandbox`)

The core orchestrator. Creates a working directory, opens the audit DB, spawns a normal host process with a sanitized environment, and waits for completion. The child can still address host resources permitted to its operating-system user unless a future isolation backend prevents it.

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

**Blocked paths:** helper-based path resolution rejects a set of sensitive dotfiles and directories. This does not constrain direct filesystem access performed by the child process.

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

**Dialer:** `Dial(network, addr)` checks policy before connecting. Only callers that explicitly use this dialer are governed by the policy; arbitrary child-process sockets are not intercepted. Blocked helper calls are logged to the audit DB. Timeout is configurable (default 10s).

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
