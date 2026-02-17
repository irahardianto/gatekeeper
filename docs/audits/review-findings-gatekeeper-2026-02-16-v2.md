# Code Audit: Gatekeeper Full Codebase (v2)
Date: 2026-02-16
Reviewer: Antigravity (fresh context)

## Summary
- **Files reviewed:** 55+ source files across 10 packages
- **Issues found:** 8 (0 critical, 3 major, 3 minor, 2 nit)
- **Test coverage:** All packages ≥85%

| Package     | Coverage | Status |
| ----------- | -------- | ------ |
| `commands`  | 89.7%    | ✅      |
| `config`    | 94.8%    | ✅      |
| `formatter` | 98.2%    | ✅      |
| `gate`      | 99.3%    | ✅      |
| `git`       | 85.5%    | ✅      |
| `llm`       | 91.7%    | ✅      |
| `parser`    | 91.3%    | ✅      |
| `pool`      | 85.3%    | ✅      |
| `runner`    | 95.5%    | ✅      |
| `logger`    | 100.0%   | ✅      |

## Critical Issues
None found. ✅

## Major Issues

- [ ] **[ARCH]** `gate.Factory` uses concrete types `*pool.Pool` and `*pool.Executor` instead of the interfaces `PoolManager` and `CommandExecutor` that `ContainerGate` depends on. This means `Factory` is tightly coupled to the production implementations and cannot be unit-tested with mock pool/executor without instantiating real types. — [factory.go:15-16](file:///home/irahardianto/works/projects/gatekeeper/internal/engine/gate/factory.go#L15-L16)
    - *Recommendation*: Change `Factory` fields to use the `PoolManager` and `CommandExecutor` interfaces already defined in `container_gate.go`. This aligns with Rule 3 (Dependency Direction) from `architectural-pattern.md`.

- [ ] **[SEC]** `container_gate.go:buildCommand` uses `fmt.Sprintf("sh '%s'", g.cfg.Path)` for script gate paths. Single-quoting alone is insufficient against paths containing single-quote characters (e.g., `my'script.sh`), which would break out of the quoted context. While the config is user-authored (low exploitation risk), defense-in-depth applies. — [container_gate.go:113](file:///home/irahardianto/works/projects/gatekeeper/internal/engine/gate/container_gate.go#L113)
    - *Recommendation*: Either reject paths containing `'` during validation in `config.go`, or use proper shell escaping (e.g., replace `'` with `'\''`).

- [ ] **[ERR]** `pipeline_orchestrator.go:80` discards the loaded `GlobalConfig` from step 2. The function calls `p.LoadGlobalConfig(ctx)` but doesn't use the returned value — it was already loaded in `pipeline.go:40-43`. This is a redundant I/O call. — [pipeline_orchestrator.go:80](file:///home/irahardianto/works/projects/gatekeeper/cmd/gatekeeper/commands/pipeline_orchestrator.go#L80)
    - *Recommendation*: Either remove the duplicate call (it's already done in `runPipeline`), or pass the `GlobalConfig` into the `Pipeline` struct so it doesn't need to be loaded twice.

## Minor Issues

- [ ] **[PAT]** `root.go` uses package-level `var` for global flag state (`flagJSON`, `flagVerbose`, etc.) which is shared mutable state. This makes tests order-dependent and prevents parallel test execution for the `commands` package. — [root.go:11-17](file:///home/irahardianto/works/projects/gatekeeper/cmd/gatekeeper/commands/root.go#L11-L17)
    - *Recommendation*: This is a well-known Cobra limitation and not easily fixable without refactoring the CLI structure. Low priority, but worth noting if parallel CLI tests become needed.

- [ ] **[PAT]** `init.go` command directly calls `os.Getwd()`, `os.ReadDir()`, `os.Stat()`, `os.MkdirAll()`, and `os.WriteFile()` without abstraction. Unlike the config package (which uses `FileSystem` interface), the init command is not testable without a real filesystem. — [init.go:25-53](file:///home/irahardianto/works/projects/gatekeeper/cmd/gatekeeper/commands/init.go#L25-L53)
    - *Recommendation*: Consider accepting a `FileSystem` interface (or similar) if test coverage for `init` becomes a priority. Currently, the smoke test covers the happy path.

- [ ] **[OBS]** `runner/progress.go:87` — `Progress.Finish()` is defined but never called anywhere in the codebase. Dead code. — [progress.go:87](file:///home/irahardianto/works/projects/gatekeeper/internal/engine/runner/progress.go#L87)
    - *Recommendation*: Either call `Finish()` at the end of `Engine.RunAll` to print the summary, or remove the method.

## Nit

- [ ] **[STYLE]** `generic.go:20` accepts a `ctx context.Context` parameter but never uses it. Consistent with the `Parser` interface, so not harmful, but the linter may flag it eventually. — [generic.go:20](file:///home/irahardianto/works/projects/gatekeeper/internal/engine/parser/generic.go#L20)

- [ ] **[DOC]** `llm_gate.go:62` discards the `skipped` diffs from `git.FilterBySize`. The skipped diffs could be logged as a debug message to help users understand why certain files were excluded from LLM review. — [llm_gate.go:62](file:///home/irahardianto/works/projects/gatekeeper/internal/engine/gate/llm_gate.go#L62)

## Verification Results
- **Format** (gofumpt): PASS ✅
- **Lint** (go vet): PASS ✅
- **Static Analysis** (staticcheck): PASS ✅
- **Security** (gosec): PASS ✅
- **Tests** (go test -race): PASS ✅ (all packages, 0 failures)
- **Build** (go build): PASS ✅
- **Coverage:** All packages ≥85.3% ✅

## Architecture Assessment

### Strengths
- **Consistent interface-based DI** across all packages (`ContainerRuntime`, `FileSystem`, `Service`, `Client`, `Parser`, `Formatter`).
- **Clean separation of concerns**: Pure business logic (filter, validate, detect, diff) separated from I/O (exec, docker, gemini).
- **Comprehensive mocks**: Every interface has a co-located test double (`mock.go` or `mock_*_test.go`).
- **Proper logging**: All operation entry points use `logger.FromContext(ctx)` with structured fields.
- **Security-conscious**: Path cleaning, gosec annotations, `SecretString` for API keys, defense-in-depth comments.
- **Retry with backoff**: LLM client implements exponential backoff with context cancellation.
- **Compile-time interface checks**: `var _ Gate = (*LLMGate)(nil)` pattern used correctly.
- **Signal handling**: Pipeline properly handles SIGINT/SIGTERM to restore git stash.

### No Issues Found In
- `config/` — Clean abstraction, proper validation, env override pattern.
- `formatter/` — Both CLI and JSON formatters are well-structured.
- `parser/` — SARIF, go-test, generic parsers all follow the interface cleanly.
- `pool/` — Proper resource cleanup, concurrent-safe with `sync.Mutex`.
- `logger/` — Simple and correct context-based logging.
- `git/` — Split into clean files by responsibility (exec, diff, hooks, stash).

## Rules Applied
- `architectural-pattern.md`: I/O Isolation, Dependency Direction, Testability
- `security-mandate.md`, `security-principles.md`: Input validation, defense-in-depth
- `logging-and-observability-mandate.md`: Operation logging coverage
- `code-organization-principles.md`: Module boundaries, function size
- `error-handling-principles.md`: Error wrapping, fail-closed patterns
- `testing-strategy.md`: Coverage targets, mock patterns
- `core-design-principles.md`: SOLID, DRY, ISP
