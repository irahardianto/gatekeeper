package pool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
)

const (
	labelManaged  = "gatekeeper.managed"
	labelPoolKey  = "gatekeeper.pool_key"
	labelImage    = "gatekeeper.image"
	labelProject  = "gatekeeper.project"
	labelLastUsed = "gatekeeper.last_used"
	labelWritable = "gatekeeper.writable"
)

// Pool manages a set of warm Docker containers.
type Pool struct {
	runtime ContainerRuntime
	mu      sync.Mutex
}

// NewPool creates a new Pool with the given runtime.
func NewPool(runtime ContainerRuntime) *Pool {
	return &Pool{
		runtime: runtime,
	}
}

// GetOrCreate returns a container ID for the given image and project path.
// If a matching warm container exists, it is returned.
// Otherwise, a new container is created and started.
func (p *Pool) GetOrCreate(ctx context.Context, img, projectPath string, writable bool) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("GetOrCreate started", "image", img, "project", projectPath, "writable", writable)

	p.mu.Lock()
	defer p.mu.Unlock()

	key := computePoolKey(img, projectPath, writable)

	// Check for existing container
	existingID, err := p.findExistingContainer(ctx, key)
	if err != nil {
		return "", fmt.Errorf("finding existing container: %w", err)
	}
	if existingID != "" {
		log.Info("GetOrCreate reused existing container", "container_id", existingID)
		return existingID, nil
	}

	// Create new container
	id, err := p.createContainer(ctx, img, projectPath, writable, key)
	if err != nil {
		return "", err
	}
	log.Info("GetOrCreate created new container", "container_id", id)
	return id, nil
}

// findExistingContainer searches for a running container with the matching pool key.
func (p *Pool) findExistingContainer(ctx context.Context, key string) (string, error) {
	opts := container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("%s=%s", labelPoolKey, key)),
			filters.Arg("status", "running"),
		),
		Limit: 1,
	}

	containers, err := p.runtime.ContainerList(ctx, opts)
	if err != nil {
		return "", err
	}

	if len(containers) > 0 {
		return containers[0].ID, nil
	}

	return "", nil
}

// createContainer pulls the image (if needed), creates, and starts a new container.
func (p *Pool) createContainer(ctx context.Context, img, projectPath string, writable bool, key string) (string, error) {
	// 1. Pull Image (lazy)
	// We use ImagePull to ensure it exists.
	logger.FromContext(ctx).Debug("pulling image", "image", img)
	reader, err := p.runtime.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("pulling image %q: %w", img, err)
	}
	if reader != nil {
		// [SEC] Verify that the image pull actually succeeded by reading the response.
		// Docker sends progress (or errors) in that stream.
		// If we don't read it, we might assume success on a failed pull.
		if _, err := io.Copy(io.Discard, reader); err != nil {
			if closeErr := reader.Close(); closeErr != nil {
				logger.FromContext(ctx).Error("failed to close image pull reader", "error", closeErr)
			}
			return "", fmt.Errorf("reading image pull response: %w", err)
		}
		if err := reader.Close(); err != nil {
			return "", fmt.Errorf("closing image pull reader: %w", err)
		}
	}

	// 2. Create Container
	config := &container.Config{
		Image:      img,
		Entrypoint: []string{"sleep", "infinity"},
		Labels: map[string]string{
			labelManaged:  "true",
			labelPoolKey:  key,
			labelImage:    img,
			labelProject:  projectPath,
			labelWritable: fmt.Sprintf("%t", writable),
			labelLastUsed: time.Now().Format(time.RFC3339),
		},
		WorkingDir: "/workspace",
	}

	if writable {
		// [SEC] Do not fallback to root on user lookup failure
		uidGid, err := getUserString()
		if err != nil {
			return "", fmt.Errorf("getting current user for writable mount: %w", err)
		}
		config.User = uidGid
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			projectMount(projectPath, writable),
			tmpMount(),
		},
	}

	resp, err := p.runtime.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	logger.FromContext(ctx).Debug("container created", "container_id", resp.ID)

	// 3. Start Container
	if err := p.runtime.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Try to remove if start fails
		_ = p.runtime.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("starting container: %w", err)
	}
	logger.FromContext(ctx).Info("container started", "container_id", resp.ID, "image", img, "project", projectPath)

	return resp.ID, nil
}

// CleanupStale removes containers that haven't been used for the given TTL.
func (p *Pool) CleanupStale(ctx context.Context, ttl time.Duration) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("CleanupStale started", "ttl", ttl)

	p.mu.Lock()
	defer p.mu.Unlock()

	opts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("%s=true", labelManaged)),
		),
	}

	containers, err := p.runtime.ContainerList(ctx, opts)
	if err != nil {
		return 0, err
	}

	count := 0
	threshold := time.Now().Add(-ttl)

	for _, c := range containers {
		lastUsedStr, ok := c.Labels[labelLastUsed]
		if !ok {
			continue
		}

		lastUsed, err := time.Parse(time.RFC3339, lastUsedStr)
		if err != nil {
			continue // Skip containers with invalid timestamps
		}

		if lastUsed.Before(threshold) {
			if err := p.runtime.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err == nil {
				count++
			} else {
				log.Error("failed to remove stale container",
					"container_id", c.ID,
					"error", err,
				)
			}
		}
	}

	log.Info("CleanupStale completed", "removed_count", count)
	return count, nil
}

// CleanupAll removes all managed containers.
func (p *Pool) CleanupAll(ctx context.Context) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("CleanupAll started")

	p.mu.Lock()
	defer p.mu.Unlock()

	opts := container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("%s=true", labelManaged)),
		),
	}

	containers, err := p.runtime.ContainerList(ctx, opts)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, c := range containers {
		if err := p.runtime.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err == nil {
			count++
		} else {
			log.Error("failed to remove container",
				"container_id", c.ID,
				"error", err,
			)
		}
	}

	log.Info("CleanupAll completed", "removed_count", count)
	return count, nil
}

func computePoolKey(image, projectPath string, writable bool) string {
	data := fmt.Sprintf("%s|%s|%t", image, projectPath, writable)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
