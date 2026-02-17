package pool

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

// ExecResult holds the result of a container execution.
type ExecResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

// Executor runs commands inside containers.
type Executor struct {
	runtime ContainerRuntime
}

// NewExecutor creates a new Executor.
func NewExecutor(runtime ContainerRuntime) *Executor {
	return &Executor{
		runtime: runtime,
	}
}

// Run executes a command inside a running container and returns the output.
// Command is wrapped in sh -c to support shell features.
// Timeout is enforced via the context.
func (e *Executor) Run(ctx context.Context, containerID, command string, timeout time.Duration) (*ExecResult, error) {
	log := logger.FromContext(ctx)
	log.Info("Executor.Run started", "container_id", containerID, "command", command, "timeout", timeout)
	start := time.Now()

	// 1. Create Exec Config
	// We wrap in sh -c to support pipes, redirects, etc.
	// Tty must be false for stdcopy to work correctly (to separate stdout/stderr).
	execConfig := container.ExecOptions{
		Cmd:          []string{"sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	execIDResp, err := e.runtime.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("creating exec: %w", err)
	}
	execID := execIDResp.ID

	// 2. Attach to Exec
	attachConfig := container.ExecAttachOptions{
		Tty: false,
	}
	resp, err := e.runtime.ContainerExecAttach(ctx, execID, attachConfig)
	if err != nil {
		return nil, fmt.Errorf("attaching to exec: %w", err)
	}
	defer resp.Close()

	// 3. Capture Output
	// Use stdcopy to demultiplex stdout and stderr.
	var stdoutBuf, stderrBuf bytes.Buffer
	outputDone := make(chan error, 1)

	go func() {
		_, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, resp.Reader)
		outputDone <- err
	}()

	// 4. Wait for completion or timeout
	// We poll check ExecInspect until it's not running, or context cancels.
	// Actually, since we attached, reading from resp.Reader until EOF should be sufficient
	// IF the process exits. But if it hangs, we need to enforce timeout.

	select {
	case err := <-outputDone:
		if err != nil {
			return nil, fmt.Errorf("reading output: %w", err)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		// Context timeout usually handles this if passed to Create/Attach,
		// but explicit timeout arg is good too.
		return nil, context.DeadlineExceeded
	}

	// 5. Inspect for Exit Code
	inspect, err := e.runtime.ContainerExecInspect(ctx, execID)
	if err != nil {
		return nil, fmt.Errorf("inspecting exec: %w", err)
	}

	result := &ExecResult{
		Stdout:   stdoutBuf.Bytes(),
		Stderr:   stderrBuf.Bytes(),
		ExitCode: inspect.ExitCode,
		Duration: time.Since(start),
	}
	log.Info("Executor.Run completed", "container_id", containerID, "exit_code", result.ExitCode, "duration", result.Duration)
	return result, nil
}
