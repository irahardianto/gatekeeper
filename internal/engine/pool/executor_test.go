package pool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// mockStream generates a Docker multiplexed stream.
// streamType: 1=stdout, 2=stderr.
func mockStream(streamType byte, content string) []byte {
	header := make([]byte, 8)
	header[0] = streamType
	length := uint32(len(content))
	binary.BigEndian.PutUint32(header[4:], length)
	return append(header, []byte(content)...)
}

func TestExecutorRun(t *testing.T) {
	ctx := context.Background()

	// Create a dummy connection for HijackedResponse
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	// Prepare multiplexed output: "hello" on stdout, "error" on stderr
	var buf bytes.Buffer
	buf.Write(mockStream(1, "hello"))
	buf.Write(mockStream(2, "error"))

	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachResp: types.HijackedResponse{
			Conn:   client,
			Reader: bufio.NewReader(&buf),
		},
		ExecInspectResp: container.ExecInspect{
			ExitCode: 0,
		},
	}

	exec := NewExecutor(mock)
	res, err := exec.Run(ctx, "container-id", "echo hello", 1*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(res.Stdout) != "hello" {
		t.Errorf("expected stdout 'hello', got %q", string(res.Stdout))
	}
	if string(res.Stderr) != "error" {
		t.Errorf("expected stderr 'error', got %q", string(res.Stderr))
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
}

func TestExecutorRun_Timeout(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()

	// Close server after 200ms so stdcopy unblocks eventually.
	go func() {
		time.Sleep(200 * time.Millisecond)
		server.Close()
	}()

	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachResp: types.HijackedResponse{
			Conn:   client,
			Reader: bufio.NewReader(client),
		},
	}

	exec := NewExecutor(mock)

	_, err := exec.Run(context.Background(), "id", "cmd", 50*time.Millisecond)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestExecutorRun_CreateError(t *testing.T) {
	mock := &MockRuntime{
		ExecCreateErr: errors.New("create failed"),
	}
	exec := NewExecutor(mock)
	_, err := exec.Run(context.Background(), "id", "cmd", 1*time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating exec") {
		t.Errorf("expected creating exec error, got %v", err)
	}
}

func TestExecutorRun_AttachError(t *testing.T) {
	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachErr:  errors.New("attach failed"),
	}
	exec := NewExecutor(mock)
	_, err := exec.Run(context.Background(), "id", "cmd", 1*time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "attaching to exec") {
		t.Errorf("expected attaching error, got %v", err)
	}
}

func TestExecutorRun_InspectError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	go func() { server.Close() }() // Close immediately to finish read

	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachResp: types.HijackedResponse{
			Conn:   client,
			Reader: bufio.NewReader(client),
		},
		ExecInspectErr: errors.New("inspect failed"),
	}
	exec := NewExecutor(mock)
	_, err := exec.Run(context.Background(), "id", "cmd", 1*time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "inspecting exec") {
		t.Errorf("expected inspecting error, got %v", err)
	}
}

func TestExecutorRun_OutputError(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachResp: types.HijackedResponse{
			Conn:   client,
			Reader: bufio.NewReader(&failReader{readErr: errors.New("read failed")}),
		},
	}
	exec := NewExecutor(mock)
	_, err := exec.Run(context.Background(), "id", "cmd", 1*time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading output") {
		t.Errorf("expected reading output error, got %v", err)
	}
}

func TestExecutorRun_ContextCancel(t *testing.T) {
	// Create a pipe that blocks so we can cancel context while it waits
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	mock := &MockRuntime{
		ExecCreateResp: container.ExecCreateResponse{ID: "exec-id"},
		ExecAttachResp: types.HijackedResponse{
			Conn:   client,
			Reader: bufio.NewReader(client),
		},
	}
	exec := NewExecutor(mock)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := exec.Run(ctx, "id", "cmd", 1*time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
