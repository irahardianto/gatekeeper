package llm

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/genai"
)

// mockGenerativeClient is a test double for GenerativeClient.
type mockGenerativeClient struct {
	responses []*genai.GenerateContentResponse
	errs      []error
	callCount int
}

func (m *mockGenerativeClient) GenerateContent(_ context.Context, _ string, _ []*genai.Content, _ *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	idx := m.callCount
	m.callCount++
	if idx < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return nil, errors.New("no more responses configured")
}

// makeResponse creates a genai response with the given text part.
func makeResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{Content: &genai.Content{Parts: []*genai.Part{
				{Text: text},
			}}},
		},
	}
}

func TestGeminiClient_Review_Success(t *testing.T) {
	mock := &mockGenerativeClient{
		responses: []*genai.GenerateContentResponse{
			makeResponse(`[{"file":"main.go","line":10,"severity":"error","message":"unused var"}]`),
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("fake-key", "test-model", factory)
	result, err := client.Review(context.Background(), "review this")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].File != "main.go" {
		t.Errorf("expected file 'main.go', got %q", result[0].File)
	}
	if result[0].Line != 10 {
		t.Errorf("expected line 10, got %d", result[0].Line)
	}
	if result[0].Tool != "test-model" {
		t.Errorf("expected tool 'test-model', got %q", result[0].Tool)
	}
}

func TestGeminiClient_Review_FactoryError(t *testing.T) {
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return nil, errors.New("factory boom")
	}

	client := NewGeminiClient("key", "", factory)
	_, err := client.Review(context.Background(), "prompt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, err) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGeminiClient_Review_RetriesOnTransientError(t *testing.T) {
	mock := &mockGenerativeClient{
		errs: []error{
			errors.New("transient failure"),
			nil,
		},
		responses: []*genai.GenerateContentResponse{
			nil,
			makeResponse(`[{"file":"a.go","line":1,"severity":"warning","message":"ok"}]`),
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("key", "m", factory)
	result, err := client.Review(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result after retry, got %d", len(result))
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 GenerateContent calls, got %d", mock.callCount)
	}
}

func TestGeminiClient_Review_AllAttemptsExhausted(t *testing.T) {
	mock := &mockGenerativeClient{
		errs: []error{
			errors.New("fail 1"),
			errors.New("fail 2"),
			errors.New("fail 3"),
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("key", "m", factory)
	_, err := client.Review(context.Background(), "prompt")
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", mock.callCount)
	}
}

func TestGeminiClient_Review_MalformedJSON(t *testing.T) {
	mock := &mockGenerativeClient{
		responses: []*genai.GenerateContentResponse{
			makeResponse(`not valid json`),
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("key", "m", factory)
	_, err := client.Review(context.Background(), "prompt")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestGeminiClient_Review_EmptyResponse(t *testing.T) {
	mock := &mockGenerativeClient{
		responses: []*genai.GenerateContentResponse{
			{Candidates: []*genai.Candidate{}},
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("key", "m", factory)
	_, err := client.Review(context.Background(), "prompt")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestGeminiClient_Review_ContextCancelled(t *testing.T) {
	mock := &mockGenerativeClient{
		errs: []error{errors.New("fail")},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := NewGeminiClient("key", "m", factory)
	_, err := client.Review(ctx, "prompt")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGeminiClient_Review_EmptyResult(t *testing.T) {
	mock := &mockGenerativeClient{
		responses: []*genai.GenerateContentResponse{
			makeResponse(`[]`),
		},
	}
	factory := func(_ context.Context, _ string) (GenerativeClient, error) {
		return mock, nil
	}

	client := NewGeminiClient("key", "m", factory)
	result, err := client.Review(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}

func TestNewGeminiClient_DefaultModel(t *testing.T) {
	client := NewGeminiClient("key", "", nil)
	if client.model != "gemini-3-pro" {
		t.Errorf("expected default model 'gemini-3-pro', got %q", client.model)
	}
	if client.factory == nil {
		t.Error("expected non-nil factory when nil is passed")
	}
}

func TestNewGeminiClient_CustomModel(t *testing.T) {
	client := NewGeminiClient("key", "custom-model", nil)
	if client.model != "custom-model" {
		t.Errorf("expected model 'custom-model', got %q", client.model)
	}
}
