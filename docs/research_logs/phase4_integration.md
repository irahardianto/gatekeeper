# Phase 4: Integration — Research Log

**Date:** 2026-02-16
**Status:** Research Complete

## 1. Current State Analysis

### 1.1 Completed Components (Phase 1-3)
- **CLI scaffold** (`cmd/gatekeeper/commands/`): All 7 commands registered as stubs with flags
- **Config** (`internal/engine/config/`): gates.yaml parser + global config with env overrides, `FileSystem` abstraction
- **Pool** (`internal/engine/pool/`): Docker warm containers, executor with stdcopy demux, preflight checks, cleanup
- **Parser** (`internal/engine/parser/`): SARIF + generic parsers, hint enrichment, `Parser` interface + `Registry`
- **Formatter** (`internal/engine/formatter/`): CLI + JSON formatters, `GateResult`/`RunResult` types
- **LLM** (`internal/engine/llm/`): Gemini client, prompt builder, line validation, `Client` interface + mock
- **Git** (`internal/engine/git/`): `Service` interface (StagedDiff, StagedFiles), `ExecService`, `MockService`
- **Logger** (`internal/platform/logger/`): Context-aware structured logging with slog

### 1.2 Missing Components (Phase 4 Scope)
1. **Gate interface** — no unified abstraction for exec/script/llm gates
2. **Parallel runner** — no engine to fan-out gate execution
3. **File filters** — no glob matching for `only`/`except`
4. **Git hooks** — no install/teardown implementation
5. **Git stash** — no stash/pop/signal handling
6. **Command wiring** — all handlers are stubs
7. **Stack detection** — no auto-generation of gates.yaml
8. **Progress indicators** — no live UI during execution

## 2. Technical Decisions

### 2.1 GateResult Location
`GateResult` currently lives in `formatter` package. The new `gate` package needs to return it.
- **Decision:** `gate/` imports `formatter.GateResult` (no circular dep since formatter doesn't import gate)
- **Alternative rejected:** Moving GateResult to a shared `types` package — unnecessary indirection

### 2.2 Package Structure
Following project-structure.md's feature-based organization:
```
internal/engine/
  gate/           # NEW: Gate interface, implementations, factory, filter
  runner/         # NEW: Parallel execution engine
  git/            # MODIFY: Add stash/hook methods to existing Service
  config/         # MODIFY: Add stack detection
  formatter/      # MODIFY: Add progress indicators
```

### 2.3 Signal Handling Pattern
- Go's `defer` does NOT run on SIGINT — requires explicit `os/signal.Notify`
- Signal handler flow: cancel ctx → wait for gates → git stash pop → os.Exit(1)
- Must install before stash, remove after stash pop

### 2.4 File Filter Glob Matching
- Go stdlib `path.Match` supports basic globs but not `**` patterns
- For MVP, use `filepath.Match` which handles `*.go`, `*.ts` patterns
- Consider `doublestar` library post-MVP if recursive globs needed

## 3. Dependencies & Import Graph

```
cmd/gatekeeper/commands/
  ├── config (load gates.yaml)
  ├── pool (container management)
  ├── gate (gate factory + execution)
  ├── runner (parallel execution)
  ├── git (stash, hooks, staged files)
  ├── formatter (output)
  └── parser (registry setup)

gate/
  ├── pool (GetOrCreate, Executor)
  ├── parser (Parse, Registry)
  ├── llm (Client)
  ├── git (StagedDiff for LLM gates)
  ├── formatter (GateResult type)
  └── config (Gate config struct)

runner/
  ├── gate (Gate interface)
  └── formatter (RunResult type)
```

No circular dependencies in this graph.

## 4. Implementation Order

Per epics document Phase 4 recommended order:
1. Gate interface (6.1) — unified abstraction
2. Parallel engine (6.2) — core runner
3. File filters (6.3) — monorepo support
4. Git hook install (1.2) — wire up trigger
5. Git stash + signal (1.3) — correctness
6. Run/dry-run (1.4) — end-to-end commands
7. Stack detection (2.3) — onboarding DX
8. Progress indicators (4.5) — UX polish
