# Phase 1: Foundation — Research Log

## Research Topics

### 1. Go CLI Framework (Cobra)
- **Library**: `github.com/spf13/cobra` (industry standard, powers kubectl, Docker CLI)
- **Structure**: `cmd/gatekeeper/main.go` → `cmd/gatekeeper/commands/*.go`
- **Pattern**: Each command in its own file; `main.go` calls `cmd.Execute()`
- **Best practice**: Use `RunE` (returns error) over `Run` for better error propagation
- **Flags**: Persistent flags for global options (`--json`, `--verbose`), local flags for command-specific
- **Go version**: 1.23+ as specified in PRD

### 2. Docker SDK for Go
- **Library**: `github.com/docker/docker/client`
- **Key APIs**: `Ping()`, `ContainerCreate()`, `ContainerStart()`, `ContainerExecCreate()`, `ContainerExecAttach()`, `ImagePull()`
- **Stream demux**: Must use `stdcopy.StdCopy` from `github.com/docker/docker/pkg/stdcopy` with `Tty=false` to correctly separate stdout/stderr. Without this, output is multiplexed with Docker headers causing corruption.
- **Pattern**: `ContainerExecCreate(Cmd: ["sh", "-c", command], Tty: false)` → `ContainerExecAttach` → `stdcopy.StdCopy(stdout, stderr, resp.Reader)` → `ExecInspect` (exit code)

### 3. SARIF Parsing
- **Library**: `github.com/owenrumney/go-sarif/v3` (supports v2.1.0 and v2.2)
- **API**: `sarif.FromBytes(data)`, `sarif.FromString(str)`, `sarif.Open(path)`
- **Schema path**: `runs[].results[]` contains result entries
- **Fields**: `ruleId`, `message.text`, `locations[0].physicalLocation` (file, line, column), `level` (error/warning/note)
- **Validation**: `report.Validate()` checks against SARIF schema

### 4. Gemini Go SDK (google.golang.org/genai)
- **Library**: `google.golang.org/genai` (maintained SDK; `github.com/google/generative-ai-go` is deprecated)
- **Structured Output**: Set `ResponseMIMEType: "application/json"` + `ResponseSchema` on `GenerateContentConfig`
- **Schema types**: `genai.TypeObject`, `genai.TypeString`, `genai.TypeArray`, `genai.TypeInteger`
- **Temperature**: Set to 0 for determinism
- **Key**: Forces API to return only valid JSON matching schema — no markdown wrapping

### 5. YAML Parsing
- **Library**: `gopkg.in/yaml.v3` (standard Go YAML library)
- **Pattern**: Define Go structs with `yaml:"field_name"` tags, `yaml.Unmarshal(data, &config)`

### 6. Structured Logging
- **Library**: `log/slog` (Go standard library since 1.21)
- **Pattern**: `slog.New(slog.NewJSONHandler(os.Stdout, &opts))` for production, `slog.NewTextHandler` for dev
- **Per rules**: Must include correlationId, operation, duration in all operation logs

### 7. Terminal Colors & Spinners
- **Colors**: `github.com/fatih/color` for ANSI colors
- **Spinner/Progress**: Custom implementation with ANSI escape codes (`\r` + cursor control)
- **TTY detection**: `github.com/mattn/go-isatty` for detecting if stdout is a terminal

### 8. Signal Handling
- **Pattern**: `os/signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)` in goroutine
- **Critical**: Go's `defer` does NOT run on `SIGINT` — must use signal trapping
- **Flow**: Signal → cancel context → wait for gates → git stash pop → os.Exit(1)

## Gotchas & Edge Cases
1. Docker `stdcopy.StdCopy` is mandatory with `Tty=false` — without it, stdout contains Docker multiplexing headers
2. `defer` doesn't run on SIGINT — needs explicit signal handler for git stash safety
3. SARIF fail-closed: malformed/empty output must be treated as system error, not pass
4. Gemini structured output mode eliminates need for regex JSON extraction
5. Container pool keys must include project path to prevent cross-project contamination
