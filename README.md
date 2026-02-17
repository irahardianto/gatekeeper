<div align="center">
  <br />
  <img src="banner.jpeg" alt="Gatekeeper" width="800" />
  <h3 align="center">Gatekeeper</h3>

  <p align="center">
    <strong>Quality gates for AI-assisted development.</strong>
    <br />
    Declarative pre-commit validation · Docker-isolated execution · AI-powered code review
  </p>

  <p align="center">
    <a href="#quick-start">Quick Start</a>
    ·
    <a href="#configuration">Configuration</a>
    ·
    <a href="#commands">Commands</a>
    ·
    <a href="https://github.com/irahardianto/gatekeeper/issues">Report Bug</a>
    ·
    <a href="https://github.com/irahardianto/gatekeeper/issues">Request Feature</a>
  </p>

  <p align="center">
    <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white" />
    <img alt="Docker" src="https://img.shields.io/badge/Docker-required-2496ED?style=flat-square&logo=docker&logoColor=white" />
    <img alt="License" src="https://img.shields.io/badge/License-MIT-green?style=flat-square" />
  </p>
</div>

<br />

## About

**Gatekeeper** is an open-source CLI tool that acts as a **git pre-commit hook gatekeeper**. It reads a declarative configuration file, executes validation gates inside Docker containers in parallel, and blocks commits that fail any gate.

Built for **AI-assisted development** — where AI agents generate code and commits at speed. Gatekeeper's structured JSON output gives agents precise `file:line:column` locations and actionable fix hints, enabling fast automated remediation. For humans, it provides rich CLI output with colors, progress indicators, and clear failure context.

### Why Gatekeeper?

| Feature                    | Benefit                                                                            |
| -------------------------- | ---------------------------------------------------------------------------------- |
| 🐳 **Container Isolation**  | All validation runs in Docker — **zero linters installed on the host**             |
| 🤖 **LLM-Powered Gates**    | Semantic code review via Gemini catches what static tools miss                     |
| 📊 **AI-Agent Output**      | Structured JSON with `file`, `line`, `hint` — minimal context, maximum signal      |
| ♻️ **Warm Containers**      | Reuses Docker containers across commits — near-instant after cold start            |
| 📐 **Unified Parsers**      | SARIF + go-test-json parsing normalizes any linter into a single error format      |
| 💡 **Enriched Hints**       | Static hint database provides actionable fix suggestions for 60+ known rules       |
| ⚡ **Parallel Execution**   | All gates run concurrently — total time ≈ slowest gate, not sum of all             |
| 🔍 **Stack Auto-Detection** | `gatekeeper init` detects Go, Node.js, and Python — generates config automatically |

---

## How It Works

```
git commit
    │
    ▼
.git/hooks/pre-commit ──▶ gatekeeper run
    │
    ├── 1. Load .gatekeeper/gates.yaml
    ├── 2. Stash unstaged changes
    ├── 3. Get staged files
    │
    ├── 4. Execute gates in parallel (Docker)
    │       ├── Gate: exec    ──▶ Container Pool ──▶ Docker Exec ──▶ Parser
    │       ├── Gate: script  ──▶ Container Pool ──▶ Docker Exec ──▶ Parser
    │       └── Gate: llm     ──▶ Gemini API ──────────────────────▶ Validator
    │
    ├── 5. Enrich hints + Aggregate results
    ├── 6. Restore stash
    │
    └── 7. Exit 0 (pass) or Exit 1 (fail)
```

---

## Quick Start

### Prerequisites

- **Go 1.25+**
- **Docker** — running and accessible (no sudo required)
- **Git** — initialized repository

### Install

**Via Go install (recommended):**
```bash
go install github.com/irahardianto/gatekeeper/cmd/gatekeeper@latest
```

**Build from source:**
```bash
git clone https://github.com/irahardianto/gatekeeper.git
cd gatekeeper
go build -o gatekeeper ./cmd/gatekeeper
sudo mv gatekeeper /usr/local/bin/
```

### Initialize

```bash
cd your-project
gatekeeper init
```

This will:
1. **Detect your stack** (Go, Node.js, Python) from marker files
2. **Generate** `.gatekeeper/gates.yaml` with sensible defaults
3. **Install** the git pre-commit hook

That's it. Your next `git commit` will run through Gatekeeper automatically.

### Try It

```bash
# Run all gates (same as what the hook does)
gatekeeper run

# Run without blocking (informational only)
gatekeeper dry-run
```

---

## Configuration

### Project Config: `.gatekeeper/gates.yaml`

```yaml
version: 1

defaults:
  container: golang:1.23       # Default container image
  timeout: 30s                 # Per-gate timeout
  blocking: true               # Gates block commits by default
  on_error: block              # System errors block by default

gates:
  - name: go-vet
    type: exec
    command: "go vet ./..."
    container: golang:1.23
    only: ["*.go"]

  - name: go-test
    type: exec
    command: "go test -race ./..."
    container: golang:1.23
    parser: go-test-json
    timeout: 120s
    only: ["*.go"]

  - name: lint
    type: exec
    command: "golangci-lint run --out-format sarif ./..."
    container: golangci/golangci-lint:latest
    parser: sarif
    only: ["*.go"]

  - name: security
    type: exec
    command: "gosec -fmt sarif ./..."
    container: securego/gosec:latest
    parser: sarif
    only: ["*.go"]

  - name: custom-check
    type: script
    path: ./scripts/validate-api.sh
    container: alpine:latest
    on_error: warn

  - name: secret-review
    type: llm
    provider: gemini-3-pro
    mode: diff
    prompt: "Check for hardcoded secrets, API keys, or credentials"
    max_file_size: 100KB
```

### User Config: `~/.config/gatekeeper/config.yaml`

```yaml
gemini_api_key: "AIza..."     # Gemini API key (never committed)
container_ttl: 5m             # Warm container TTL
```

### Environment Variables

| Variable                | Overrides             |
| ----------------------- | --------------------- |
| `GATEKEEPER_GEMINI_KEY` | `gemini_api_key`      |
| `GATEKEEPER_TTL`        | `container_ttl`       |
| `GATEKEEPER_NO_COLOR`   | `output.color: false` |

---

## Gate Types

### `exec` — Run a command

Execute any command inside a Docker container. The project root is mounted at `/workspace`.

```yaml
- name: lint
  type: exec
  command: "golangci-lint run --out-format sarif ./..."
  container: golangci/golangci-lint:latest
  parser: sarif
  only: ["*.go"]
```

### `script` — Run a local script

Execute a shell script from your project inside a container.

```yaml
- name: validate-api
  type: script
  path: ./scripts/validate-api.sh
  container: alpine:latest
```

### `llm` — AI-powered review

Send staged diffs to Gemini for semantic code review. Catches issues static tools miss — secrets, logic errors, security anti-patterns.

```yaml
- name: secret-review
  type: llm
  provider: gemini-3-pro
  prompt: "Check for hardcoded secrets, API keys, or credentials"
  max_file_size: 100KB
  blocking: false           # Advisory — don't block commits
```

> **Note**: LLM gates require a Gemini API key in your user config or `GATEKEEPER_GEMINI_KEY` environment variable. Use `--skip-llm` to skip all LLM gates.

---

## Gate Options

| Field           | Type     | Default              | Description                                             |
| --------------- | -------- | -------------------- | ------------------------------------------------------- |
| `name`          | string   | *required*           | Unique gate identifier                                  |
| `type`          | string   | *required*           | `exec`, `script`, or `llm`                              |
| `command`       | string   | —                    | Command to run (`exec` type)                            |
| `path`          | string   | —                    | Script path (`script` type)                             |
| `container`     | string   | `defaults.container` | Docker image                                            |
| `parser`        | string   | `generic`            | Output parser: `sarif`, `go-test-json`, or `generic`    |
| `timeout`       | duration | `30s`                | Maximum execution time                                  |
| `blocking`      | bool     | `true`               | Whether failure blocks the commit                       |
| `on_error`      | string   | `block`              | System error policy: `block` or `warn`                  |
| `only`          | []string | —                    | Only run if staged files match these globs              |
| `except`        | []string | —                    | Skip if staged files match these globs                  |
| `writable`      | bool     | `false`              | Mount project read-write (for tools that need to write) |
| `provider`      | string   | —                    | LLM provider (`llm` type)                               |
| `prompt`        | string   | —                    | Review instructions (`llm` type)                        |
| `max_file_size` | string   | —                    | Skip files larger than this (`llm` type)                |

---

## Commands

| Command               | Description                                            |
| --------------------- | ------------------------------------------------------ |
| `gatekeeper init`     | Detect stack, generate config, install pre-commit hook |
| `gatekeeper run`      | Execute all gates — exit 1 if any blocking gate fails  |
| `gatekeeper dry-run`  | Execute all gates — always exit 0 (informational)      |
| `gatekeeper teardown` | Remove the pre-commit hook (config preserved)          |
| `gatekeeper cleanup`  | Stop and remove all Gatekeeper Docker containers       |
| `gatekeeper version`  | Print version, Go version, and build info              |

### Global Flags

| Flag            | Description                                      |
| --------------- | ------------------------------------------------ |
| `--json`        | Output results as structured JSON to stdout      |
| `--verbose`     | Include raw tool stdout/stderr in output         |
| `--no-color`    | Disable colored output                           |
| `--fail-fast`   | Cancel remaining gates on first blocking failure |
| `--skip <name>` | Skip specific gates by name                      |
| `--skip-llm`    | Skip all LLM gates                               |

---

## Output

### CLI Output

```
🔒 Gatekeeper — 1 failed, 3 passed (2.3s)

  ✅ go-test        1.2s
  ✅ go-vet         0.8s
  ❌ security       0.5s
  ✅ secret-review  2.1s

──────────────────────────────────────

❌ security (gosec)

  auth/handler.go:45:12  error  G101
  Potential hardcoded credentials
  💡 Use environment variables or a secret manager

  api/db.go:23:5  warning  G201
  SQL string formatting
  💡 Use parameterized queries ($1, $2)

──────────────────────────────────────

❌ Commit blocked — 1 gate failed
```

### JSON Output (`--json`)

Designed for AI agents and CI pipelines — every error includes precise location and actionable hints:

```json
{
  "passed": false,
  "duration_ms": 2300,
  "gates": [
    {
      "name": "security",
      "type": "exec",
      "passed": false,
      "blocking": true,
      "duration_ms": 500,
      "errors": [
        {
          "file": "auth/handler.go",
          "line": 45,
          "column": 12,
          "severity": "error",
          "rule": "gosec:G101",
          "message": "Potential hardcoded credentials",
          "hint": "Use environment variables or a secret manager",
          "tool": "gosec"
        }
      ]
    }
  ]
}
```

---

## Parsers

Gatekeeper normalizes output from any tool into a unified `StructuredError` format.

| Parser         | Use Case                                             | Source                             |
| -------------- | ---------------------------------------------------- | ---------------------------------- |
| `sarif`        | Universal — most modern linters support SARIF output | golangci-lint, gosec, ruff, ESLint |
| `go-test-json` | Go test output in JSON format                        | `go test -json`                    |
| `generic`      | Fallback — uses exit code + raw output               | Any tool                           |

The **hint enrichment system** provides actionable fix suggestions for 60+ known rule IDs across Go (gosec, staticcheck, vet), JavaScript (ESLint), and Python (ruff, flake8, bandit).

---

## Architecture

```
gatekeeper/
├── cmd/gatekeeper/           # CLI entry point (Cobra)
│   └── commands/             # run, dry-run, init, teardown, cleanup, version
└── internal/
    ├── engine/               # Core engine (designed as reusable library)
    │   ├── config/           # gates.yaml + global config parsing + stack detection
    │   ├── gate/             # Gate interface + exec, script, LLM implementations
    │   ├── runner/           # Parallel execution engine with progress tracking
    │   ├── pool/             # Docker container pool (warm runners, TTL cleanup)
    │   ├── parser/           # SARIF, go-test-json, generic parsers + hint database
    │   ├── formatter/        # CLI + JSON output formatters
    │   ├── llm/              # Gemini client, prompt builder, response validation
    │   └── git/              # Stash, staged files, hook management, diff extraction
    └── platform/
        └── logger/           # Structured logging
```

**Key design principles:**
- **Testability-first**: All I/O behind interfaces — Docker, Git, LLM, filesystem
- **Fail-closed**: Malformed parser output = system error, never a silent pass
- **Stateless**: Container labels are the source of truth — no local state files
- **Signal-safe**: `SIGINT`/`SIGTERM` trapping guarantees stash restoration

---

## Roadmap

- [x] Core CLI with `run`, `dry-run`, `init`, `teardown`, `cleanup`
- [x] Docker container pool with warm runners
- [x] SARIF + go-test-json + generic parsers
- [x] LLM-powered gates (Gemini)
- [x] Stack auto-detection (Go, Node.js, Python)
- [x] Parallel execution with fail-fast
- [x] Enriched hint database (60+ rules)
- [ ] MCP Server — expose engine as MCP tools for real-time AI agent validation
- [ ] Multi-provider LLM — OpenAI, Anthropic, Ollama support
- [ ] Blessed images — pre-built `gatekeeper/go`, `gatekeeper/node`, `gatekeeper/python`
- [ ] Shadow mode — gates report but never block (team onboarding)
- [ ] LLM cache — cache results by diff hash to reduce API calls
- [ ] `gatekeeper add` — interactive gate wizard / preset library
- [ ] Host execution — `--host-exec` fallback for Docker-less environments

---

## Contributing

Contributions are welcome! Here's how to get started:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

```bash
# Build
go build -o gatekeeper ./cmd/gatekeeper

# Test
go test -race ./...

# Lint
go vet ./...
```

---

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with ❤️ for AI-assisted development
</p>
