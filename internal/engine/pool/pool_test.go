package pool

import (
	"context"
	"errors"
	"io"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func TestGetOrCreate_Existing(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{ID: "existing-id", Labels: map[string]string{labelPoolKey: "dummy"}},
		},
	}
	p := NewPool(mock)

	id, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != "existing-id" {
		t.Errorf("expected existing ID 'existing-id', got %q", id)
	}
}

func TestGetOrCreate_New(t *testing.T) {
	mock := &MockRuntime{
		ListResp:        []container.Summary{}, // No existing
		ImagePullReader: io.NopCloser(strings.NewReader("pulling...")),
		CreateResp:      container.CreateResponse{ID: "new-id"},
	}
	p := NewPool(mock)

	id, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != "new-id" {
		t.Errorf("expected new ID 'new-id', got %q", id)
	}
}

func TestCleanupStale(t *testing.T) {
	now := time.Now()

	mock := &MockRuntime{
		ListResp: []container.Summary{
			{
				ID: "stale-1",
				Labels: map[string]string{
					labelManaged:  "true",
					labelLastUsed: now.Add(-10 * time.Minute).Format(time.RFC3339),
				},
			},
			{
				ID: "fresh-1",
				Labels: map[string]string{
					labelManaged:  "true",
					labelLastUsed: now.Add(-1 * time.Minute).Format(time.RFC3339),
				},
			},
		},
	}
	p := NewPool(mock)

	count, err := p.CleanupStale(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mock remove always succeeds, so we count successful removal attempts.
	if count != 1 {
		t.Errorf("expected 1 removed, got %d", count)
	}
}

func TestCleanupAll(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{ID: "c1", Labels: map[string]string{labelManaged: "true"}},
			{ID: "c2", Labels: map[string]string{labelManaged: "true"}},
		},
	}
	p := NewPool(mock)

	count, err := p.CleanupAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 removed, got %d", count)
	}
}

func TestProjectMount_RO(t *testing.T) {
	m := projectMount("/path", false)
	if m.Type != mount.TypeBind {
		t.Errorf("expected bind mount, got %s", m.Type)
	}
	if !m.ReadOnly {
		t.Errorf("expected read-only")
	}
	if m.Source != "/path" {
		t.Errorf("expected source /path, got %s", m.Source)
	}
	if m.Target != "/workspace" {
		t.Errorf("expected target /workspace, got %s", m.Target)
	}
}

func TestProjectMount_RW(t *testing.T) {
	m := projectMount("/path", true)
	if m.ReadOnly {
		t.Errorf("expected read-write")
	}
}

func TestTmpMount(t *testing.T) {
	m := tmpMount()
	if m.Type != mount.TypeTmpfs {
		t.Errorf("expected tmpfs mount, got %s", m.Type)
	}
	if m.Target != "/tmp" {
		t.Errorf("expected target /tmp, got %s", m.Target)
	}
}

func TestGetOrCreate_ImagePullError(t *testing.T) {
	mock := &MockRuntime{
		ListResp:     []container.Summary{},
		ImagePullErr: io.ErrUnexpectedEOF,
	}

	p := NewPool(mock)
	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pulling image") {
		t.Errorf("expected pulling image error, got %v", err)
	}
}

func TestGetOrCreate_ContainerCreateError(t *testing.T) {
	mock := &MockRuntime{
		ListResp:        []container.Summary{},
		ImagePullReader: io.NopCloser(strings.NewReader("ok")),
		CreateErr:       io.ErrClosedPipe,
	}
	p := NewPool(mock)
	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating container") {
		t.Errorf("expected creating container error, got %v", err)
	}
}

func TestGetOrCreate_StartError(t *testing.T) {
	mock := &MockRuntime{
		ListResp:        []container.Summary{},
		ImagePullReader: io.NopCloser(strings.NewReader("ok")),
		CreateResp:      container.CreateResponse{ID: "created-id"},
		StartErr:        io.ErrClosedPipe,
	}
	p := NewPool(mock)
	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "starting container") {
		t.Errorf("expected starting container error, got %v", err)
	}
}

func TestCleanupStale_RemoveError(t *testing.T) {
	// Create a container that is considered stale
	staleTime := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{
				ID: "stale-id",
				Labels: map[string]string{
					"gatekeeper.managed":   "true",
					"gatekeeper.last_used": staleTime,
				},
			},
		},
		RemoveErr: errors.New("remove failed"),
	}

	p := NewPool(mock)
	count, err := p.CleanupStale(context.Background(), 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be 0 because remove failed
	if count != 0 {
		t.Errorf("expected 0 removed, got %d", count)
	}
}

func TestCleanupAll_RemoveError(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{
				ID: "id-1",
				Labels: map[string]string{
					"gatekeeper.managed": "true",
				},
			},
		},
		RemoveErr: errors.New("remove failed"),
	}

	p := NewPool(mock)
	count, err := p.CleanupAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be 0 because remove failed
	if count != 0 {
		t.Errorf("expected 0 removed, got %d", count)
	}
}

func TestGetOrCreate_Writable(t *testing.T) {
	mock := &MockRuntime{
		ListResp:        []container.Summary{},
		ImagePullReader: io.NopCloser(strings.NewReader("pulling...")),
		CreateResp:      container.CreateResponse{ID: "writable-id"},
	}
	p := NewPool(mock)

	// Mock userCurrent for success
	original := userCurrent
	defer func() { userCurrent = original }()
	userCurrent = func() (*user.User, error) {
		return &user.User{Uid: "1000", Gid: "1000"}, nil
	}

	id, err := p.GetOrCreate(context.Background(), "alpine", "/proj", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "writable-id" {
		t.Errorf("expected writable-id, got %q", id)
	}
}

type failReader struct {
	readErr  error
	closeErr error
}

func (f *failReader) Read(p []byte) (n int, err error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return 0, io.EOF
}

func (f *failReader) Close() error {
	return f.closeErr
}

func TestGetOrCreate_ImagePullReaderError(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{},
		ImagePullReader: &failReader{
			readErr: errors.New("read failed"),
		},
	}
	p := NewPool(mock)

	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading image pull response") {
		t.Errorf("expected reading error, got %v", err)
	}
}

func TestGetOrCreate_ImagePullReaderCloseError(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{},
		ImagePullReader: &failReader{
			closeErr: errors.New("close failed"),
		},
	}
	p := NewPool(mock)

	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "closing image pull reader") {
		t.Errorf("expected closing error, got %v", err)
	}
}

func TestGetOrCreate_ImagePullReaderReadAndCloseError(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{},
		ImagePullReader: &failReader{
			readErr:  errors.New("read failed"),
			closeErr: errors.New("close also failed"),
		},
	}
	p := NewPool(mock)

	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Primary error should be about reading
	if !strings.Contains(err.Error(), "reading image pull response") {
		t.Errorf("expected reading error, got %v", err)
	}
}

func TestGetOrCreate_ListError(t *testing.T) {
	mock := &MockRuntime{
		ListErr: errors.New("list failed"),
	}
	p := NewPool(mock)

	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "finding existing container") {
		t.Errorf("expected finding existing container error, got %v", err)
	}
}

func TestCleanupStale_ListError(t *testing.T) {
	mock := &MockRuntime{
		ListErr: errors.New("list failed"),
	}
	p := NewPool(mock)

	_, err := p.CleanupStale(context.Background(), 5*time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCleanupAll_ListError(t *testing.T) {
	mock := &MockRuntime{
		ListErr: errors.New("list failed"),
	}
	p := NewPool(mock)

	_, err := p.CleanupAll(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetOrCreate_WritableUserError(t *testing.T) {
	mock := &MockRuntime{
		ListResp:        []container.Summary{},
		ImagePullReader: io.NopCloser(strings.NewReader("ok")),
		CreateResp:      container.CreateResponse{ID: "id"},
	}
	p := NewPool(mock)

	original := userCurrent
	defer func() { userCurrent = original }()
	userCurrent = func() (*user.User, error) {
		return nil, errors.New("user lookup failed")
	}

	_, err := p.GetOrCreate(context.Background(), "alpine", "/proj", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting current user") {
		t.Errorf("expected user error, got %v", err)
	}
}

func TestCleanupStale_InvalidTimestamp(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{
				ID: "bad-ts",
				Labels: map[string]string{
					labelManaged:  "true",
					labelLastUsed: "not-a-timestamp",
				},
			},
		},
	}
	p := NewPool(mock)

	count, err := p.CleanupStale(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be 0 — container skipped due to invalid timestamp
	if count != 0 {
		t.Errorf("expected 0 removed, got %d", count)
	}
}

func TestCleanupStale_NoLastUsedLabel(t *testing.T) {
	mock := &MockRuntime{
		ListResp: []container.Summary{
			{
				ID: "no-label",
				Labels: map[string]string{
					labelManaged: "true",
				},
			},
		},
	}
	p := NewPool(mock)

	count, err := p.CleanupStale(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 removed, got %d", count)
	}
}
