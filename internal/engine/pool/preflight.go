package pool

import (
	"context"
	"fmt"
	"strings"
)

// PreflightError wraps a Docker connectivity error with a user-friendly message.
type PreflightError struct {
	Hint  string
	Cause error
}

func (e *PreflightError) Error() string {
	return fmt.Sprintf("❌ %s", e.Hint)
}

func (e *PreflightError) Unwrap() error {
	return e.Cause
}

// CheckDocker verifies the Docker daemon is available.
// Returns a PreflightError with context-specific hints on failure (NFR9).
// This check must run BEFORE git stash to avoid leaving git in a stashed state.
func CheckDocker(ctx context.Context, runtime ContainerRuntime) error {
	err := runtime.Ping(ctx)
	if err == nil {
		return nil
	}

	return classifyDockerError(err)
}

// classifyDockerError inspects the error message to produce actionable user hints.
func classifyDockerError(err error) *PreflightError {
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "permission denied"):
		return &PreflightError{
			Hint:  "Docker permission denied. Run: sudo usermod -aG docker $USER, then re-login.",
			Cause: err,
		}
	case strings.Contains(msg, "connection refused"):
		return &PreflightError{
			Hint:  "Docker is not running. Start it with: sudo systemctl start docker",
			Cause: err,
		}
	case strings.Contains(msg, "no such file or directory") || strings.Contains(msg, "not found"):
		return &PreflightError{
			Hint:  "Docker is required but not found. Install it from https://docker.com",
			Cause: err,
		}
	default:
		return &PreflightError{
			Hint:  "Docker is required but not found. Install it from https://docker.com",
			Cause: err,
		}
	}
}
