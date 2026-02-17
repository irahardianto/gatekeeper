# Research Log: Remaining Development Gaps

Date: 2026-02-16

## Scope

Review all 23 stories across 6 epics against the current implementation to identify what's missing.

## Findings

### Story Completion Status

| Epic                         | Stories                 | Status                              |
| ---------------------------- | ----------------------- | ----------------------------------- |
| Epic 1: CLI & Git            | 1.1, 1.2, 1.3, 1.4      | ✅ All implemented                   |
| Epic 2: Configuration        | 2.1, 2.2, 2.3           | ✅ All implemented                   |
| Epic 3: Container Pool       | 3.1, 3.2, 3.3, 3.4      | ✅ All implemented                   |
| Epic 4: Parsers & Output     | 4.1, 4.2, 4.3, 4.4, 4.5 | ⚠️ go-test-json parser missing (4.1) |
| Epic 5: LLM Gate             | 5.1, 5.2, 5.3           | ✅ All implemented                   |
| Epic 6: Runner & Integration | 6.1, 6.2, 6.3, 6.4      | ✅ All implemented                   |

### Coverage Report (2026-02-16)

| Package     | Coverage | Target | Gap                            |
| ----------- | -------- | ------ | ------------------------------ |
| `commands`  | 61.6%    | 85%    | ❌ Large gap — composition root |
| `config`    | 94.8%    | 85%    | ✅                              |
| `formatter` | 98.2%    | 85%    | ✅                              |
| `gate`      | 99.3%    | 85%    | ✅                              |
| `git`       | 82.8%    | 85%    | ⚠️ Slight gap                   |
| `llm`       | 91.7%    | 85%    | ✅                              |
| `parser`    | 92.0%    | 85%    | ✅                              |
| `pool`      | 85.3%    | 85%    | ✅                              |
| `runner`    | 95.5%    | 85%    | ✅                              |
| `logger`    | 100.0%   | 85%    | ✅                              |

### Audit v2 Status

Several v2 findings already resolved:
- **ARCH**: ExecGate/ScriptGate merged into `ContainerGate` ✅
- **ERR**: `ErrGatesFailed` sentinel error in place ✅
- **SEC**: Script path quoting in `buildCommand()` ✅
- **OBS**: `gemini.go` uses `logger.FromContext(ctx)` ✅
- **PAT**: `init.go` uses `strings.Join` ✅
- **PAT**: `FilterGates` behavior documented ✅
- **TEST**: `gate` coverage at 99.3% ✅

Remaining:
- **PAT**: Duplicate `// 13.` step numbers in `pipeline.go`
- **TEST**: `commands` coverage at 61.6% (composition root)

### go test -json Format

TestEvent struct: `Time`, `Action`, `Package`, `Test`, `Elapsed`, `Output`, `FailedBuild`.
Key actions: `start`, `run`, `pass`, `fail`, `output`, `skip`.
Format: newline-separated JSON objects (not a JSON array).

## Technologies Researched

- `go test -json` output format (TestEvent specification from `cmd/test2json`)
- Existing parser patterns in codebase (SarifParser, GenericParser)
