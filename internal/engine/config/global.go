package config

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"gopkg.in/yaml.v3"
)

// SecretString is a string that is redacted when printed.
type SecretString string

func (s SecretString) String() string {
	return "[REDACTED]"
}

func (s SecretString) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// IsEmpty returns true if the secret string is empty.
func (s SecretString) IsEmpty() bool {
	return string(s) == ""
}

// GlobalConfig holds user-level settings that persist across projects.
type GlobalConfig struct {
	GeminiAPIKey  SecretString  `yaml:"gemini_api_key"`
	ContainerTTL  time.Duration `yaml:"container_ttl"`
	OutputColor   bool          `yaml:"-"` // derived from Output.Color
	OutputVerbose bool          `yaml:"-"` // derived from Output.Verbose
	Output        OutputConfig  `yaml:"output"`
}

// OutputConfig holds output-related user preferences.
type OutputConfig struct {
	Color   *bool `yaml:"color"`
	Verbose *bool `yaml:"verbose"`
}

const defaultContainerTTL = 5 * time.Minute

// LoadGlobalConfig reads user-level configuration from ~/.config/gatekeeper/config.yaml.
// If the file does not exist, default values are returned (not an error).
// Environment variables override file values.
func (l *Loader) LoadGlobalConfig(ctx context.Context) (*GlobalConfig, error) {
	home, err := l.fs.UserHomeDir()
	if err != nil {
		// Cannot determine home directory — use defaults.
		cfg := defaultGlobalConfig()
		log := logger.FromContext(ctx)
		applyEnvOverrides(cfg, l.getenv, log)
		return cfg, nil
	}
	path := filepath.Join(home, ".config", "gatekeeper", "config.yaml")
	return l.LoadGlobalConfigFrom(ctx, path)
}

// LoadGlobalConfigFrom reads user-level configuration from a specific path.
// If the file does not exist, default values are returned (not an error).
// Environment variables override file values.
func (l *Loader) LoadGlobalConfigFrom(ctx context.Context, path string) (*GlobalConfig, error) {
	log := logger.FromContext(ctx)
	log.Debug("loading global config", "path", path)
	cfg := defaultGlobalConfig()

	// [SEC] Clean path
	path = filepath.Clean(path)

	data, err := l.fs.ReadFile(path)
	if err != nil {
		if l.fs.IsNotExist(err) {
			applyEnvOverrides(cfg, l.getenv, log)
			return cfg, nil
		}
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	if cfg.Output.Color != nil {
		cfg.OutputColor = *cfg.Output.Color
	}
	if cfg.Output.Verbose != nil {
		cfg.OutputVerbose = *cfg.Output.Verbose
	}

	applyEnvOverrides(cfg, l.getenv, log)

	return cfg, nil
}

// LoadGlobalConfig reads user-level configuration using the real file system.
func LoadGlobalConfig(ctx context.Context) (*GlobalConfig, error) {
	return NewLoader(&RealFileSystem{}).LoadGlobalConfig(ctx)
}

// LoadGlobalConfigFrom reads user-level configuration from a specific path using the real file system.
func LoadGlobalConfigFrom(ctx context.Context, path string) (*GlobalConfig, error) {
	return NewLoader(&RealFileSystem{}).LoadGlobalConfigFrom(ctx, path)
}

func defaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		ContainerTTL: defaultContainerTTL,
		OutputColor:  true,
	}
}

// applyEnvOverrides applies environment variable overrides to the config.
// The getenv parameter abstracts os.Getenv for testability.
// The log parameter provides structured logging consistent with the rest of the codebase.
func applyEnvOverrides(cfg *GlobalConfig, getenv func(string) string, log *slog.Logger) {
	if key := getenv("GATEKEEPER_GEMINI_KEY"); key != "" {
		cfg.GeminiAPIKey = SecretString(key)
	}

	if ttlStr := getenv("GATEKEEPER_TTL"); ttlStr != "" {
		d, err := time.ParseDuration(ttlStr)
		if err != nil {
			log.Warn("invalid GATEKEEPER_TTL value, using default", "value", ttlStr, "error", err)
		} else {
			cfg.ContainerTTL = d
		}
	}

	if noColor := getenv("GATEKEEPER_NO_COLOR"); noColor != "" {
		// Any truthy value disables color.
		noColor = strings.ToLower(noColor)
		if noColor == "1" || noColor == "true" || noColor == "yes" {
			cfg.OutputColor = false
		}
	}
}
