package pool

import (
	"fmt"
	"os/user"

	"github.com/docker/docker/api/types/mount"
)

// projectMount returns a mount configuration for the project root.
// writable determines if the mount is read-write or read-only.
// If writable is true, the container user is mapped to the host user to avoid permission issues.
func projectMount(path string, writable bool) mount.Mount {
	m := mount.Mount{
		Type:   mount.TypeBind,
		Source: path,
		Target: "/workspace",
	}

	m.ReadOnly = !writable

	return m
}

// tmpMount returns a configuration for an ephemeral /tmp directory.
func tmpMount() mount.Mount {
	return mount.Mount{
		Type:   mount.TypeTmpfs,
		Target: "/tmp",
	}
}

// userCurrent is a variable to allow mocking in tests.
var userCurrent = user.Current

// getUserString returns the "uid:gid" string for the current user.
func getUserString() (string, error) {
	u, err := userCurrent()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", u.Uid, u.Gid), nil
}
