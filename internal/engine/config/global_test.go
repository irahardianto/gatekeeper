package config

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

func TestLoadGlobalConfig_ValidFile(t *testing.T) {
	mockFS := NewMockFileSystem()
	path := "/config.yaml"
	mockFS.Files[path] = []byte(`
gemini_api_key: "test-key-123"
container_ttl: 10m
output:
  color: false
  verbose: true
`)

	loader := NewLoader(mockFS)
	cfg, err := loader.LoadGlobalConfigFrom(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GeminiAPIKey != "test-key-123" {
		t.Errorf("expected GeminiAPIKey 'test-key-123', got %q", cfg.GeminiAPIKey)
	}
	if cfg.ContainerTTL != 10*time.Minute {
		t.Errorf("expected ContainerTTL 10m, got %v", cfg.ContainerTTL)
	}
	if cfg.OutputColor {
		t.Error("expected OutputColor false")
	}
	if !cfg.OutputVerbose {
		t.Error("expected OutputVerbose true")
	}
}

func TestLoadGlobalConfig_MissingFile(t *testing.T) {
	mockFS := NewMockFileSystem()
	loader := NewLoader(mockFS)

	cfg, err := loader.LoadGlobalConfigFrom(context.Background(), "/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("missing file should not error, got: %v", err)
	}

	// Should use defaults
	if cfg.ContainerTTL != defaultContainerTTL {
		t.Errorf("expected default TTL %v, got %v", defaultContainerTTL, cfg.ContainerTTL)
	}
	if !cfg.OutputColor {
		t.Error("expected default OutputColor true")
	}
}

func TestLoadGlobalConfig_EnvOverrides(t *testing.T) {
	mockFS := NewMockFileSystem()
	path := "/config.yaml"
	mockFS.Files[path] = []byte(`
gemini_api_key: "file-key"
container_ttl: 10m
`)

	// Set env vars that should override file values.
	t.Setenv("GATEKEEPER_GEMINI_KEY", "env-key-456")
	t.Setenv("GATEKEEPER_TTL", "3m")
	t.Setenv("GATEKEEPER_NO_COLOR", "1")

	loader := NewLoader(mockFS)
	cfg, err := loader.LoadGlobalConfigFrom(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GeminiAPIKey != "env-key-456" {
		t.Errorf("expected env-overridden GeminiAPIKey 'env-key-456', got %q", cfg.GeminiAPIKey)
	}
	if cfg.ContainerTTL != 3*time.Minute {
		t.Errorf("expected env-overridden TTL 3m, got %v", cfg.ContainerTTL)
	}
	if cfg.OutputColor {
		t.Error("expected OutputColor false due to GATEKEEPER_NO_COLOR=1")
	}
}

func TestLoadGlobalConfig_EnvOverridesNoFile(t *testing.T) {
	t.Setenv("GATEKEEPER_GEMINI_KEY", "only-env-key")

	mockFS := NewMockFileSystem()
	loader := NewLoader(mockFS)

	cfg, err := loader.LoadGlobalConfigFrom(context.Background(), "/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GeminiAPIKey != "only-env-key" {
		t.Errorf("expected GeminiAPIKey from env, got %q", cfg.GeminiAPIKey)
	}
}

func TestLoadGlobalConfig_InvalidTTLEnv(t *testing.T) {
	mockFS := NewMockFileSystem()
	path := "/config.yaml"
	mockFS.Files[path] = []byte(`container_ttl: 5m`)

	t.Setenv("GATEKEEPER_TTL", "not-a-duration")

	loader := NewLoader(mockFS)
	cfg, err := loader.LoadGlobalConfigFrom(context.Background(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid TTL env should be ignored, file value preserved.
	if cfg.ContainerTTL != 5*time.Minute {
		t.Errorf("expected TTL from file (5m), got %v", cfg.ContainerTTL)
	}
}

func TestLoadGlobalConfig_NoColorVariants(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"1", false},
		{"true", false},
		{"TRUE", false},
		{"yes", false},
		{"YES", false},
		{"0", true},     // not a truthy value → color stays on
		{"false", true}, // not a truthy value → color stays on
		{"", true},      // empty → color stays on
	}

	for _, tt := range tests {
		t.Run("GATEKEEPER_NO_COLOR="+tt.envValue, func(t *testing.T) {
			t.Setenv("GATEKEEPER_NO_COLOR", tt.envValue)

			mockFS := NewMockFileSystem()
			loader := NewLoader(mockFS)

			cfg, err := loader.LoadGlobalConfigFrom(context.Background(), "/nonexistent/config.yaml")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.OutputColor != tt.expected {
				t.Errorf("with NO_COLOR=%q, expected OutputColor=%v, got %v", tt.envValue, tt.expected, cfg.OutputColor)
			}
		})
	}
}

func TestSecretString_Redaction(t *testing.T) {
	s := SecretString("my-secret-key")
	if s.String() != "[REDACTED]" {
		t.Errorf("expected redacted string, got %q", s.String())
	}
}

func TestSecretString_MarshalYAML(t *testing.T) {
	s := SecretString("my-secret-key")
	val, err := s.MarshalYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %q", val)
	}
}

func TestLoadGlobalConfig_UserHomeDirError(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.UserHomeErr = errors.New("no home dir")
	loader := NewLoader(mockFS)

	cfg, err := loader.LoadGlobalConfig(context.Background())
	if err != nil {
		t.Fatalf("expected nil error (defaults used), got: %v", err)
	}
	if cfg.ContainerTTL != defaultContainerTTL {
		t.Errorf("expected default TTL, got %v", cfg.ContainerTTL)
	}
}

func TestLoadGlobalConfig_ReadError(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.UserHome = "/home/test"
	mockFS.ReadErrors["/home/test/.config/gatekeeper/config.yaml"] = errors.New("disk error")

	loader := NewLoader(mockFS)
	_, err := loader.LoadGlobalConfig(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadGlobalConfig_InvalidYAML(t *testing.T) {
	mockFS := NewMockFileSystem()
	// Use content that yaml.Unmarshal rejects when deserializing into GlobalConfig.
	// A tab character at the start is invalid YAML.
	mockFS.Files["/config.yaml"] = []byte("\t: invalid")

	loader := NewLoader(mockFS)
	_, err := loader.LoadGlobalConfigFrom(context.Background(), "/config.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApplyEnvOverrides_CustomGetenv(t *testing.T) {
	cfg := defaultGlobalConfig()
	getenv := func(key string) string {
		switch key {
		case "GATEKEEPER_GEMINI_KEY":
			return "custom-key"
		case "GATEKEEPER_TTL":
			return "7m"
		case "GATEKEEPER_NO_COLOR":
			return "true"
		}
		return ""
	}

	applyEnvOverrides(cfg, getenv, slog.Default())

	if cfg.GeminiAPIKey != "custom-key" {
		t.Errorf("expected custom-key, got %q", cfg.GeminiAPIKey)
	}
	if cfg.ContainerTTL != 7*time.Minute {
		t.Errorf("expected 7m, got %v", cfg.ContainerTTL)
	}
	if cfg.OutputColor {
		t.Error("expected OutputColor false")
	}
}

func TestLoadGlobalConfigConvenience(t *testing.T) {
	// Convenience function uses RealFileSystem — just ensure it runs without panic.
	cfg, err := LoadGlobalConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadGlobalConfigFromConvenience(t *testing.T) {
	// Convenience function with non-existent path should return defaults.
	cfg, err := LoadGlobalConfigFrom(context.Background(), "/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ContainerTTL != defaultContainerTTL {
		t.Errorf("expected default TTL, got %v", cfg.ContainerTTL)
	}
}
