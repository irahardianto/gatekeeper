# Research Log: Logging & Test Coverage

**Date:** 2026-02-16
**Status:** In Progress

## 1. Current State Analysis

### 1.1 Project Structure
- `pkg/engine` has been migrated to `internal/engine`.
- `cmd/gatekeeper` contains the CLI entry point.
- No structured logging library exists; `log.Printf` and `fmt.Errorf` are used.

### 1.2 Logging Status
- **Audit Finding:** Zero observability.
- **Verification:**
  - `internal/engine/config`: No logging.
  - `internal/engine/pool`: Uses `log.Printf` for errors in cleanup.
  - `cmd/gatekeeper`: Flag `--verbose` exists but likely just prints raw output, no structured logger setup.

### 1.3 Test Coverage Status
- **Audit Findings:**
  - `config`: 68.1% (Low). Missing tests for `LoadGlobalConfig` and real file loading.
  - `pool`: 73.6% (Low). Missing error paths for container lifecycle.
- **Verification:**
  - `config_test.go`: Tests logical config loading with mock FS. `Load` with real FS is untested. `LoadGlobalConfig` not tested in `config_test.go` (need to check `global.go`).
  - `pool_test.go`: Tests happy path for `GetOrCreate` and `Cleanup`. No tests for docker client failures (pull fail, create fail).

## 2. Implementation Strategy

### 2.1 Structured Logging
- **Library:** `log/slog` (Standard Library).
- **Pattern:**
  - Create `internal/platform/logger` package.
  - Initialize in `cmd/gatekeeper`.
  - Inject via `context.Context` to support Correlation IDs (as requested by audit).
  - Add `Logger(ctx)` helper.

### 2.2 Test Coverage Improvements
- **Config:**
  - Refactor `LoadGlobalConfig` to use `FileSystem` interface or similar abstraction for `os.ReadFile` and `os.UserHomeDir`.
  - Add tests for `LoadGlobalConfig` with mocked home dir and file content.
- **Pool:**
  - Enhance `MockRuntime` to allow error injection (e.g. `FailPull`, `FailCreate`).
  - Add unit tests for `GetOrCreate` failure scenarios.

## 3. Risks & Dependencies
- **Refactoring:** Changing `LoadGlobalConfig` signature might affect CLI entry usage.
- **Context Propagation:** Ensuring `ctx` is passed everywhere for logging.
