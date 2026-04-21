# AGENTS.md — spec-cli Development Guidelines

## Project Context

`spec` is a developer control plane CLI written in Go. See `SPEC.md` for full product specification. This file defines coding standards for all contributors and AI agents working on this codebase.

## Architecture Rules

- **`cmd/` is thin.** Parse flags, resolve config, call `internal/`. Zero business logic in command files.
- **Engines depend on interfaces, never concrete adapters.** All adapter interfaces live in `internal/adapter/`. Implementations live in `internal/adapter/<provider>/`. Engines import `internal/adapter`, never `internal/adapter/github`.
- **Only `internal/git/` shells out to git.** No other package calls `exec.Command("git", ...)`.
- **Only `internal/store/` touches SQLite.** No other package opens the database or writes raw SQL.
- **Adapters are isolated.** A broken Confluence adapter must not affect GitHub operations. Adapter failures return errors; callers degrade gracefully.
- **Noop adapters exist for every category.** Unconfigured integrations use noop adapters that return empty results and nil errors. Never panic on missing config.

## Go Standards

### Structure

- Standard Go project layout: `cmd/`, `internal/`, `main.go`.
- One file per concern. Prefer many small files over few large ones.
- Package names are short, singular nouns: `config`, `markdown`, `pipeline`, `store`, `sync`, `build`.
- No `util`, `helper`, or `common` packages. Find a real name or put it where it's used.

### Naming

- Follow [Effective Go](https://go.dev/doc/effective_go) naming conventions.
- Exported names are descriptive: `ReadMeta`, `WithSpecsRepo`, `ReplaceSection`.
- Unexported names can be shorter when scope is clear.
- Interfaces describe behaviour: `CommsAdapter`, `AIAdapter`. Don't prefix with `I`.
- Error variables: `ErrSpecNotFound`, `ErrGateNotMet`, `ErrSyncConflict`.

### Functions

- Functions do one thing. If you're writing a comment to separate "phases" inside a function, extract them.
- Max ~50 lines per function. If it's longer, decompose it.
- Accept interfaces, return structs.
- Context (`context.Context`) is the first parameter on anything that does I/O or may be cancelled.
- Return `error` as the last return value. Wrap errors with context: `fmt.Errorf("advancing %s: %w", specID, err)`.

### Error Handling

- Handle every error. No `_ = someFunc()` unless there's a comment explaining why.
- Wrap errors with `%w` for unwrapping. Add the operation context, not just the error.
- User-facing errors must include the next action: not `"config not found"` but `"config not found — run 'spec config init' to set up"`.
- Adapter errors are never fatal. Return the error; let the caller decide whether to degrade or abort.

### Testing

- Table-driven tests for any function with more than two interesting inputs.
- Golden file tests for the markdown engine: input `.md` → expected parsed output.
- Test against interfaces, not implementations. Use test doubles that implement adapter interfaces.
- Each test creates its own state (in-memory SQLite via `:memory:`, temp directories via `t.TempDir()`). No shared test fixtures.
- Test names describe the scenario: `TestAdvance_GateNotMet_ReturnsError`, `TestSyncInbound_RoleGuard_IgnoresMismatch`.
- Run `go vet` and `golangci-lint` clean. No lint exceptions without a comment explaining why.

## Design Principles

### KISS

- Solve the problem in front of you, not the hypothetical future one.
- Choose boring, obvious solutions over clever ones. If a `map` and a `for` loop work, don't reach for a framework.
- No premature abstraction. Extract an interface when you have two concrete implementations, not before.

### Modularity & Loose Coupling

- Packages communicate through exported interfaces and types, not by reaching into each other's internals.
- The adapter registry is the only place that knows which concrete adapter matches which provider string. Everything else works with interfaces.
- Config is loaded once at startup and injected into engines. Engines never read config files directly.
- The AI service layer returns `(string, error)` or `(nil, nil)`. Callers always handle the nil case — AI is never a hard dependency.

### Robustness

- Degrade, don't crash. If GitHub is unreachable, the dashboard shows cached data. If the AI provider is down, `spec draft` tells the user and exits cleanly.
- All network calls have timeouts. Default: 10s for API calls, 30s for git operations.
- `WithSpecsRepo` retries on push conflict (up to 3 times). Every retry is logged.
- SQLite writes use transactions. No partial state on error.

### Readability

- Code should be self-documenting. If the name and structure don't explain what's happening, refactor before adding a comment.
- Comments explain *why*, not *what*. `// Retry because concurrent push may have advanced the ref` is good. `// increment i` is noise.
- Exported functions and types always have a doc comment.
- No magic numbers. Use named constants.
- Consistent ordering within files: types, then constructors, then methods, then helpers.

## Dependency Rules

- No CGo. The binary must be statically linked and cross-compilable.
- Minimise dependencies. Every new `go get` should be justified — prefer the standard library when it's within 20% of the effort.
- Pin major versions. Use Go module version suffixes (`/v2`, `/v62`).
- `modernc.org/sqlite` for SQLite (pure Go). Not `mattn/go-sqlite3`.

## Commit & PR Conventions

- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- One logical change per commit. Refactors and features don't share a commit.
- PR descriptions reference the spec: `Implements US-12 (Dashboard)` or `Addresses §7.9 Build Engine`.
- PRs follow the stack plan in `SPEC.md §7.14`. Dependencies between PRs are explicit.
