package pool

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerRuntime implements ContainerRuntime using the Docker SDK.
type DockerRuntime struct {
	client client.APIClient
}

// NewDockerRuntimeFrom creates a DockerRuntime with the given API client.
// Use this constructor when you need to inject a specific client (e.g., for testing).
func NewDockerRuntimeFrom(cli client.APIClient) *DockerRuntime {
	return &DockerRuntime{client: cli}
}

// NewDockerRuntime creates a DockerRuntime with a Docker SDK client.
// Uses the default Docker host and API version negotiation.
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return NewDockerRuntimeFrom(cli), nil
}

// Ping checks if the Docker daemon is available and responsive.
func (d *DockerRuntime) Ping(ctx context.Context) error {
	_, err := d.client.Ping(ctx)
	return err
}

// ImagePull requests the Docker host to pull an image from a remote registry.
func (d *DockerRuntime) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	return d.client.ImagePull(ctx, ref, options)
}

// ContainerCreate creates a new container based on the given configuration.
func (d *DockerRuntime) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string) (container.CreateResponse, error) {
	return d.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerStart sends a request to the Docker daemon to start a container.
func (d *DockerRuntime) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return d.client.ContainerStart(ctx, containerID, options)
}

// ContainerInspect returns low-level information about a container.
func (d *DockerRuntime) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return d.client.ContainerInspect(ctx, containerID)
}

// ContainerList returns the list of containers in the Docker host.
func (d *DockerRuntime) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return d.client.ContainerList(ctx, options)
}

// ContainerRemove kills and removes a container from the Docker host.
func (d *DockerRuntime) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return d.client.ContainerRemove(ctx, containerID, options)
}

// ContainerExecCreate sets up an exec instance in a container.
func (d *DockerRuntime) ContainerExecCreate(ctx context.Context, ctr string, config container.ExecOptions) (container.ExecCreateResponse, error) {
	return d.client.ContainerExecCreate(ctx, ctr, config)
}

// ContainerExecAttach attaches a connection to an exec process in a container.
func (d *DockerRuntime) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	return d.client.ContainerExecAttach(ctx, execID, config)
}

// ContainerExecInspect returns information about a specific exec process on the Docker host.
func (d *DockerRuntime) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return d.client.ContainerExecInspect(ctx, execID)
}
