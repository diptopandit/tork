# Contributing to tork

Contributions are welcome.

## Process

1. Open an issue for significant changes
2. Fork and create a feature branch
3. Add or update tests
4. Run full checks locally
5. Submit a PR with a clear description and rationale

## Development Setup

```bash
git clone https://github.com/diptopandit/tork.git
cd tork
go build ./...
go test ./...
go vet ./...
```

Run binaries:

```bash
go run ./cmd/tork-cli --help
go run ./cmd/tork
```

## Project Architecture

Layered design:

```
Interface Layer (CLI/TUI)
  → Application Layer (use-cases/services)
    → Domain Layer (entities/ports)
      → Infrastructure Layer (SQLite/config/logger/search)
```

Repository layout:

```text
cmd/
  tork-cli/
  tork/

internal/
  application/
  domain/
  infrastructure/
    config/
    db/
    logger/
    repository/
    search/
  interface/
    cli/
    tui/
      views/
```

## PR Quality Expectations

- Keep changes focused
- Maintain layer boundaries
- Include tests for behavior changes
- Keep docs in sync with implementation

## Commit Prefixes

- `feat` — new feature
- `fix` — bug fix
- `refactor` — code restructure
- `test` — test additions or changes
- `docs` — documentation updates

## Release Readiness Checklist

Before announcing a public release, ensure:
- License file is added and approved
- Version tags and changelog process are defined
- Issue templates and PR template are added
- Security disclosure path is documented
- Example screenshots or terminal recordings are added
- CI status badge reflects real CI workflow
