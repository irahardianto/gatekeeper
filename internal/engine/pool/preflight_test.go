package pool

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCheckDocker_Available(t *testing.T) {
	mock := &MockRuntime{PingErr: nil}

	err := CheckDocker(context.Background(), mock)
	if err != nil {
		t.Fatalf("expected no error when Docker is available, got: %v", err)
	}
}

func TestCheckDocker_PermissionDenied(t *testing.T) {
	mock := &MockRuntime{
		PingErr: errors.New("Got permission denied while trying to connect to the Docker daemon socket"),
	}

	err := CheckDocker(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error for permission denied")
	}

	var pErr *PreflightError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected PreflightError, got %T", err)
	}

	if !strings.Contains(pErr.Hint, "permission denied") {
		t.Errorf("expected permission denied hint, got: %s", pErr.Hint)
	}
	if !strings.Contains(pErr.Hint, "usermod") {
		t.Errorf("expected usermod fix suggestion, got: %s", pErr.Hint)
	}
}

func TestCheckDocker_ConnectionRefused(t *testing.T) {
	mock := &MockRuntime{
		PingErr: errors.New("Cannot connect to the Docker daemon. Is the docker daemon running? connection refused"),
	}

	err := CheckDocker(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}

	var pErr *PreflightError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected PreflightError, got %T", err)
	}

	if !strings.Contains(pErr.Hint, "not running") {
		t.Errorf("expected 'not running' hint, got: %s", pErr.Hint)
	}
	if !strings.Contains(pErr.Hint, "systemctl") {
		t.Errorf("expected systemctl fix suggestion, got: %s", pErr.Hint)
	}
}

func TestCheckDocker_NotFound(t *testing.T) {
	mock := &MockRuntime{
		PingErr: errors.New("docker: no such file or directory"),
	}

	err := CheckDocker(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error for Docker not found")
	}

	var pErr *PreflightError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected PreflightError, got %T", err)
	}

	if !strings.Contains(pErr.Hint, "docker.com") {
		t.Errorf("expected docker.com install hint, got: %s", pErr.Hint)
	}
}

func TestCheckDocker_UnknownError(t *testing.T) {
	mock := &MockRuntime{
		PingErr: errors.New("some unexpected error from Docker"),
	}

	err := CheckDocker(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error for unknown Docker failure")
	}

	var pErr *PreflightError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected PreflightError, got %T", err)
	}

	// Unknown errors fall back to generic install suggestion.
	if !strings.Contains(pErr.Hint, "docker.com") {
		t.Errorf("expected fallback docker.com hint, got: %s", pErr.Hint)
	}
}

func TestPreflightError_Unwrap(t *testing.T) {
	original := errors.New("original error")
	pErr := &PreflightError{
		Hint:  "some hint",
		Cause: original,
	}

	if !errors.Is(pErr, original) {
		t.Error("expected Unwrap to expose original error")
	}
}

func TestPreflightError_Error(t *testing.T) {
	pErr := &PreflightError{
		Hint:  "Do something",
		Cause: errors.New("original"),
	}

	expected := "❌ Do something"
	if pErr.Error() != expected {
		t.Errorf("expected %q, got %q", expected, pErr.Error())
	}
}
