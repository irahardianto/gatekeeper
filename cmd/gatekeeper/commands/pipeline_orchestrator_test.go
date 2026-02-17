package commands

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/irahardianto/gatekeeper/internal/engine/config"
	"github.com/irahardianto/gatekeeper/internal/engine/formatter"
	"github.com/irahardianto/gatekeeper/internal/engine/gate"
	"github.com/irahardianto/gatekeeper/internal/engine/git"
)

// --- Mock implementations ---

type mockDockerChecker struct {
	err error
}

func (m *mockDockerChecker) CheckDocker(_ context.Context) error {
	return m.err
}

type mockGateCreator struct {
	gates []gate.Gate
	err   error
}

func (m *mockGateCreator) CreateAll(_ []config.Gate) ([]gate.Gate, error) {
	return m.gates, m.err
}

type mockGateRunner struct {
	result *formatter.RunResult
	err    error
}

func (m *mockGateRunner) RunAll(_ context.Context, _ []gate.Gate, _ bool, _ []string) (*formatter.RunResult, error) {
	return m.result, m.err
}

type mockGitService struct {
	stagedFiles         []string
	stagedFilesErr      error
	stashed             bool
	stashErr            error
	stashPopErr         error
	cleanWritableErr    error
	stashPopCalled      bool
	cleanWritableCalled bool
}

func (m *mockGitService) StagedDiff(_ context.Context) ([]git.FileDiff, error) {
	return nil, nil
}

func (m *mockGitService) StagedFiles(_ context.Context) ([]string, error) {
	return m.stagedFiles, m.stagedFilesErr
}

func (m *mockGitService) InstallHook(_ context.Context) error { return nil }
func (m *mockGitService) RemoveHook(_ context.Context) error  { return nil }

func (m *mockGitService) Stash(_ context.Context) (bool, error) {
	return m.stashed, m.stashErr
}

func (m *mockGitService) StashPop(_ context.Context) error {
	m.stashPopCalled = true
	return m.stashPopErr
}

func (m *mockGitService) CleanWritableFiles(_ context.Context) error {
	m.cleanWritableCalled = true
	return m.cleanWritableErr
}

// --- stubGate implements gate.Gate ---

type stubGate struct{}

func (s *stubGate) Execute(_ context.Context) (*formatter.GateResult, error) {
	return &formatter.GateResult{Name: "stub", Passed: true}, nil
}

// --- Helpers ---

func defaultConfig() *config.GatekeeperConfig {
	blocking := true
	return &config.GatekeeperConfig{
		Version: 1,
		Gates: []config.Gate{
			{Name: "lint", Type: config.GateTypeExec, Container: "golangci/golangci-lint", Command: "golangci-lint run", Blocking: &blocking},
		},
	}
}

func passingRunResult() *formatter.RunResult {
	return &formatter.RunResult{
		Passed:     true,
		DurationMs: 100,
		Gates: []formatter.GateResult{
			{Name: "lint", Passed: true, Blocking: true},
		},
	}
}

func failingRunResult() *formatter.RunResult {
	return &formatter.RunResult{
		Passed:     false,
		DurationMs: 200,
		Gates: []formatter.GateResult{
			{Name: "lint", Passed: false, Blocking: true},
		},
	}
}

func newTestPipeline(gitSvc *mockGitService) (*Pipeline, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	return &Pipeline{
		Git:    gitSvc,
		Docker: &mockDockerChecker{},
		Gates:  &mockGateCreator{gates: []gate.Gate{&stubGate{}}},
		Runner: &mockGateRunner{result: passingRunResult()},
		LoadConfig: func(_ context.Context, _ string) (*config.GatekeeperConfig, error) {
			return defaultConfig(), nil
		},
		GlobalConfig: &config.GlobalConfig{},
		ConfigPath:   "/fake/gates.yaml",
		Stdout:       stdout,
		Stderr:       stderr,
		stagedFiles:  []string{"main.go"},
	}, stdout, stderr
}

// --- Tests ---

func TestPipeline_AllGatesPass(t *testing.T) {
	gitSvc := &mockGitService{}
	p, stdout, _ := newTestPipeline(gitSvc)

	err := p.Execute(context.Background(), PipelineOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("expected formatted output on stdout")
	}
}

func TestPipeline_GatesFail(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.Runner = &mockGateRunner{result: failingRunResult()}

	err := p.Execute(context.Background(), PipelineOpts{})
	if !errors.Is(err, ErrGatesFailed) {
		t.Fatalf("expected ErrGatesFailed, got %v", err)
	}
}

func TestPipeline_DryRun_GatesFail(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.Runner = &mockGateRunner{result: failingRunResult()}

	err := p.Execute(context.Background(), PipelineOpts{DryRun: true})
	if err != nil {
		t.Fatalf("dry run should not return error even on gate failure, got %v", err)
	}
}

func TestPipeline_ConfigLoadError(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.LoadConfig = func(_ context.Context, _ string) (*config.GatekeeperConfig, error) {
		return nil, errors.New("config not found")
	}

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "config not found" {
		t.Errorf("expected 'config not found', got %q", err.Error())
	}
}

func TestPipeline_DockerCheckFails(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.Docker = &mockDockerChecker{err: errors.New("docker not running")}

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "docker not running" {
		t.Errorf("expected 'docker not running', got %q", err.Error())
	}
}

func TestPipeline_StashAndRestore(t *testing.T) {
	gitSvc := &mockGitService{stashed: true}
	p, _, _ := newTestPipeline(gitSvc)

	err := p.Execute(context.Background(), PipelineOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !gitSvc.stashPopCalled {
		t.Error("expected StashPop to be called after execution")
	}
}

func TestPipeline_NoGatesAfterFilter(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, stderr := newTestPipeline(gitSvc)

	// Skip the only gate.
	err := p.Execute(context.Background(), PipelineOpts{Skip: []string{"lint"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("No gates to run")) {
		t.Errorf("expected 'No gates to run' message, got %q", stderr.String())
	}
}

func TestPipeline_WritableCleanup(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	// Override config to include a writable gate.
	p.LoadConfig = func(_ context.Context, _ string) (*config.GatekeeperConfig, error) {
		blocking := true
		return &config.GatekeeperConfig{
			Version: 1,
			Gates: []config.Gate{
				{Name: "format", Type: config.GateTypeExec, Container: "golang", Command: "gofmt -w .", Blocking: &blocking, Writable: true},
			},
		}, nil
	}

	err := p.Execute(context.Background(), PipelineOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !gitSvc.cleanWritableCalled {
		t.Error("expected CleanWritableFiles to be called for writable gate")
	}
}

func TestPipeline_GlobalConfigNil(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.GlobalConfig = nil

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "global config not loaded" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestPipeline_StashError(t *testing.T) {
	gitSvc := &mockGitService{stashErr: errors.New("stash failed")}
	p, _, _ := newTestPipeline(gitSvc)

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "stashing changes: stash failed" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestPipeline_GateCreatorError(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.Gates = &mockGateCreator{err: errors.New("factory error")}

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "factory error" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestPipeline_RunnerError(t *testing.T) {
	gitSvc := &mockGitService{}
	p, _, _ := newTestPipeline(gitSvc)
	p.Runner = &mockGateRunner{err: errors.New("execution error")}

	err := p.Execute(context.Background(), PipelineOpts{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "execution error" {
		t.Errorf("unexpected error: %q", err.Error())
	}
}

func TestPipeline_JSONOutput(t *testing.T) {
	gitSvc := &mockGitService{}
	p, stdout, _ := newTestPipeline(gitSvc)

	err := p.Execute(context.Background(), PipelineOpts{JSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON output should contain "passed" field.
	if !bytes.Contains(stdout.Bytes(), []byte(`"passed"`)) {
		t.Errorf("expected JSON output, got %q", stdout.String())
	}
}
