package git

import (
	"context"
)

// MockService is a test double for git.Service.
type MockService struct {
	Diffs       []FileDiff
	DiffErr     error
	Files       []string
	FilesErr    error
	HookInstErr error
	HookRemErr  error
	StashDone   bool
	StashErr    error
	PopErr      error
	CleanErr    error
}

// StagedDiff returns the configured diffs.
func (m *MockService) StagedDiff(_ context.Context) ([]FileDiff, error) {
	return m.Diffs, m.DiffErr
}

// StagedFiles returns the configured file list.
func (m *MockService) StagedFiles(_ context.Context) ([]string, error) {
	return m.Files, m.FilesErr
}

// InstallHook returns the configured error.
func (m *MockService) InstallHook(_ context.Context) error {
	return m.HookInstErr
}

// RemoveHook returns the configured error.
func (m *MockService) RemoveHook(_ context.Context) error {
	return m.HookRemErr
}

// Stash returns the configured stash result.
func (m *MockService) Stash(_ context.Context) (bool, error) {
	return m.StashDone, m.StashErr
}

// StashPop returns the configured error.
func (m *MockService) StashPop(_ context.Context) error {
	return m.PopErr
}

// CleanWritableFiles returns the configured error.
func (m *MockService) CleanWritableFiles(_ context.Context) error {
	return m.CleanErr
}
