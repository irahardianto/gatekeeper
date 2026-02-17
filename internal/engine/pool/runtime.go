// Package pool manages Docker container lifecycle for gatekeeper gate execution.
package pool

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerRuntime abstracts Docker operations for testability (NFR10).
// Production code uses DockerRuntime; tests use MockRuntime.
type ContainerRuntime interface {
	// Ping checks if the Docker daemon is available and responsive.
	Ping(ctx context.Context) error

	// ImagePull requests the docker host to pull an image from a remote registry.
	ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error)

	// ContainerCreate creates a new container based on the given configuration.
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *v1.Platform, name string) (container.CreateResponse, error)

	// ContainerStart starts a container.
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error

	// ContainerInspect returns the container information.
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)

	// ContainerList returns the list of containers in the docker host.
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)

	// ContainerRemove kills and removes a container from the docker host.
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error

	// ContainerExecCreate creates a new exec configuration to run an exec process.
	ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (container.ExecCreateResponse, error)

	// ContainerExecAttach attaches a connection to an exec process.
	ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)

	// ContainerExecInspect returns information about a specific exec process.
	ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error)
}
