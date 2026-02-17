// Package config handles parsing and validation of gatekeeper configuration files.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"gopkg.in/yaml.v3"
)

// GateType represents the type of a gate.
type GateType string

const (
	GateTypeExec   GateType = "exec"
	GateTypeScript GateType = "script"
	GateTypeLLM    GateType = "llm"
)

// OnErrorPolicy defines behavior when a system error occurs.
type OnErrorPolicy string

const (
	OnErrorBlock OnErrorPolicy = "block"
	OnErrorWarn  OnErrorPolicy = "warn"
)

// ErrConfigNotFound is returned when the config file does not exist.
var ErrConfigNotFound = errors.New("no .gatekeeper/gates.yaml found. Run 'gatekeeper init' first")

// GatekeeperConfig is the top-level project configuration.
type GatekeeperConfig struct {
	Version  int      `yaml:"version"`
	Defaults Defaults `yaml:"defaults"`
	Gates    []Gate   `yaml:"gates"`
}

// Defaults holds default values that are applied to gates missing optional fields.
type Defaults struct {
	Container string        `yaml:"container"`
	Timeout   time.Duration `yaml:"timeout"`
	Blocking  *bool         `yaml:"blocking"`
	OnError   OnErrorPolicy `yaml:"on_error"`
	FailFast  bool          `yaml:"fail_fast"`
}

// Gate represents a single validation gate configuration.
type Gate struct {
	Name        string        `yaml:"name"`
	Type        GateType      `yaml:"type"`
	Command     string        `yaml:"command,omitempty"`
	Path        string        `yaml:"path,omitempty"`
	Container   string        `yaml:"container,omitempty"`
	Parser      string        `yaml:"parser,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty"`
	Blocking    *bool         `yaml:"blocking,omitempty"`
	OnError     OnErrorPolicy `yaml:"on_error,omitempty"`
	Only        []string      `yaml:"only,omitempty"`
	Except      []string      `yaml:"except,omitempty"`
	Writable    bool          `yaml:"writable,omitempty"`
	Provider    string        `yaml:"provider,omitempty"`
	Mode        string        `yaml:"mode,omitempty"`
	Prompt      string        `yaml:"prompt,omitempty"`
	MaxFileSize string        `yaml:"max_file_size,omitempty"`
}

// IsBlocking returns whether this gate blocks commits on failure.
// Falls back to true if not explicitly set.
func (g *Gate) IsBlocking() bool {
	if g.Blocking != nil {
		return *g.Blocking
	}
	return true
}

// GetOnError returns the on_error policy, defaulting to "block".
func (g *Gate) GetOnError() OnErrorPolicy {
	if g.OnError != "" {
		return g.OnError
	}
	return OnErrorBlock
}

// Loader handles loading configuration from the file system.
type Loader struct {
	fs     FileSystem
	getenv func(string) string
}

// NewLoader creates a new Loader with the given file system.
// Uses os.Getenv for environment variable lookups by default.
func NewLoader(fs FileSystem) *Loader {
	return &Loader{fs: fs, getenv: os.Getenv}
}

// NewLoaderWithEnv creates a Loader with a custom getenv function for testability.
func NewLoaderWithEnv(fs FileSystem, getenv func(string) string) *Loader {
	return &Loader{fs: fs, getenv: getenv}
}

// Load reads and parses a gates.yaml configuration file from the given path.
// Returns ErrConfigNotFound if the file does not exist.
func (l *Loader) Load(ctx context.Context, path string) (*GatekeeperConfig, error) {
	logger.FromContext(ctx).Debug("loading config file", "path", path)
	// [SEC] Prevent path traversal
	path = filepath.Clean(path)

	data, err := l.fs.ReadFile(path)
	if err != nil {
		if l.fs.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg GatekeeperConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing gates.yaml: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Load reads and parses a gates.yaml configuration file from the given path using the real file system.
// Returns ErrConfigNotFound if the file does not exist.
func Load(ctx context.Context, path string) (*GatekeeperConfig, error) {
	return NewLoader(&RealFileSystem{}).Load(ctx, path)
}

// applyDefaults applies values from the defaults section to gates missing optional fields.
func applyDefaults(cfg *GatekeeperConfig) {
	for i := range cfg.Gates {
		g := &cfg.Gates[i]

		if g.Container == "" && cfg.Defaults.Container != "" {
			g.Container = cfg.Defaults.Container
		}
		if g.Timeout == 0 && cfg.Defaults.Timeout > 0 {
			g.Timeout = cfg.Defaults.Timeout
		}
		if g.Blocking == nil && cfg.Defaults.Blocking != nil {
			val := *cfg.Defaults.Blocking
			g.Blocking = &val
		}
		if g.OnError == "" && cfg.Defaults.OnError != "" {
			g.OnError = cfg.Defaults.OnError
		}
	}
}

// validate checks that all gates have required fields for their type.
// Returns a joined error if multiple gates have issues, so users can fix all at once.
func validate(cfg *GatekeeperConfig) error {
	var errs []error
	for _, g := range cfg.Gates {
		if g.Name == "" {
			errs = append(errs, fmt.Errorf("gate at position has missing required field 'name'"))
			continue
		}

		switch g.Type {
		case GateTypeExec:
			if g.Command == "" {
				errs = append(errs, fmt.Errorf("gate %q: missing required field 'command' for type 'exec'", g.Name))
			}
		case GateTypeScript:
			if g.Path == "" {
				errs = append(errs, fmt.Errorf("gate %q: missing required field 'path' for type 'script'", g.Name))
			} else if strings.Contains(g.Path, "'") {
				errs = append(errs, fmt.Errorf("gate %q: path contains invalid character (single quote)", g.Name))
			}
		case GateTypeLLM:
			if g.Provider == "" {
				errs = append(errs, fmt.Errorf("gate %q: missing required field 'provider' for type 'llm'", g.Name))
			}
			if g.Prompt == "" {
				errs = append(errs, fmt.Errorf("gate %q: missing required field 'prompt' for type 'llm'", g.Name))
			}
		case "":
			errs = append(errs, fmt.Errorf("gate %q: missing required field 'type'", g.Name))
		default:
			errs = append(errs, fmt.Errorf("gate %q: unknown gate type %q (valid: exec, script, llm)", g.Name, g.Type))
		}
	}

	return errors.Join(errs...)
}
