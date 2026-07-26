# Contributing to Vault

Thanks for your interest in contributing. Vault is a sandbox for AI agents, built with Go and SQLite.

## Ways to Contribute

- **Bug fixes** - Check issues labeled `bug`
- **Features** - Check issues labeled `enhancement` or `good first issue`
- **Injection patterns** - Add new prompt injection detection patterns
- **Network policy** - Improve host filtering and connection logging
- **Filesystem** - Enhance overlay isolation and path allowlisting
- **Docs** - Improve README, add examples, write guides
- **Tests** - Add test coverage for sandbox and policy packages

## Setup

```bash
git clone https://github.com/valtors/vault.git
cd vault
go mod tidy
go build ./cmd/vault
```

## Development

```bash
# Build
go build ./cmd/vault

# Run tests with race detector
go test ./internal/... -race -count=1

# Run a sandboxed command
./vault run -- echo hello

# Start the API server
./vault serve -port 9090
```

## AI Agent Contribution Guide

If you use AI tools to contribute, document which tools you used and which parts they generated. Keep human review in the loop.

## License

By contributing, you agree that your contributions will be licensed under the MIT license.
