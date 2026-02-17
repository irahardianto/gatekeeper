package commands

import (
	"io/fs"
	"os"
)

// osInitFS implements InitFS using the os package.
type osInitFS struct{}

func (o *osInitFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (o *osInitFS) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (o *osInitFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (o *osInitFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (o *osInitFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm) // #nosec G306 -- config file, not sensitive
}
