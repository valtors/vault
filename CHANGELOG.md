# Changelog

## v0.1.0 - 2026-07-30

### Added
- Experimental policy and audit environment for AI agents
- Helper-level filesystem allow/deny path rules (not an OS isolation boundary)
- Advisory resource configuration and process timeout
- Permission-gated tool access (read, write, execute)
- Audit log of all agent actions
- SQLite-backed audit trail
- Race-free concurrent access tested with -race flag
- 83 tests, 76.3% coverage
- CI pipeline (with -race)
- ARCHITECTURE.md
- SECURITY.md, CODEOWNERS, issue/PR templates
- GitHub Discussions enabled
