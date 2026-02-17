package pool

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MockRuntime is a test double for ContainerRuntime.
type MockRuntime struct {
	PingErr         error
	ImagePullErr    error
	ImagePullReader io.ReadCloser
	CreateResp      container.CreateResponse
	CreateErr       error
	StartErr        error
	InspectResp     container.InspectResponse
	InspectErr      error
	ListResp        []container.Summary
	ListErr         error
	RemoveErr       error
	ExecCreateResp  container.ExecCreateResponse
	ExecCreateErr   error
	ExecAttachResp  types.HijackedResponse
	ExecAttachErr   error
	ExecInspectResp container.ExecInspect
	ExecInspectErr  error
}

func (m *MockRuntime) Ping(_ context.Context) error {
	return m.PingErr
}

func (m *MockRuntime) ImagePull(_ context.Context, _ string, _ image.PullOptions) (io.ReadCloser, error) {
	return m.ImagePullReader, m.ImagePullErr
}

func (m *MockRuntime) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, _ *network.NetworkingConfig, _ *v1.Platform, _ string) (container.CreateResponse, error) {
	return m.CreateResp, m.CreateErr
}

func (m *MockRuntime) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return m.StartErr
}

func (m *MockRuntime) ContainerInspect(_ context.Context, _ string) (container.InspectResponse, error) {
	return m.InspectResp, m.InspectErr
}

func (m *MockRuntime) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return m.ListResp, m.ListErr
}

func (m *MockRuntime) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	return m.RemoveErr
}

func (m *MockRuntime) ContainerExecCreate(_ context.Context, _ string, _ container.ExecOptions) (container.ExecCreateResponse, error) {
	return m.ExecCreateResp, m.ExecCreateErr
}

func (m *MockRuntime) ContainerExecAttach(_ context.Context, _ string, _ container.ExecAttachOptions) (types.HijackedResponse, error) {
	return m.ExecAttachResp, m.ExecAttachErr
}

func (m *MockRuntime) ContainerExecInspect(_ context.Context, _ string) (container.ExecInspect, error) {
	return m.ExecInspectResp, m.ExecInspectErr
}

// MockPool is a test double for Pool.
type MockPool struct {
	ContainerID string
	Err         error
}

func (m *MockPool) GetOrCreate(_ context.Context, _, _ string, _ bool) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.ContainerID, nil
}

func (m *MockPool) CleanupStale(_ context.Context, _ time.Duration) (int, error) {
	return 0, nil
}

func (m *MockPool) CleanupAll(_ context.Context) (int, error) {
	return 0, nil
}

// MockExecutor is a test double for Executor.
type MockExecutor struct {
	Result      *ExecResult
	Err         error
	LastTimeout time.Duration
}

func (m *MockExecutor) Run(_ context.Context, _, _ string, timeout time.Duration) (*ExecResult, error) {
	m.LastTimeout = timeout
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Result, nil
}
