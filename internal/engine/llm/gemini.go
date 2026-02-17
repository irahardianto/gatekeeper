package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/irahardianto/gatekeeper/internal/engine/parser"
	"github.com/irahardianto/gatekeeper/internal/platform/logger"
	"google.golang.org/genai"
)

// GenerativeClient abstracts the Gemini generative AI client for testability.
type GenerativeClient interface {
	// GenerateContent sends a prompt and returns a response.
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

// ClientFactory creates a GenerativeClient. Production code uses DefaultClientFactory;
// tests inject a factory that returns a mock.
type ClientFactory func(ctx context.Context, apiKey string) (GenerativeClient, error)

// genaiClient wraps the real genai.Client to satisfy GenerativeClient.
type genaiClient struct {
	inner *genai.Client
}

func (g *genaiClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return g.inner.Models.GenerateContent(ctx, model, contents, config)
}

// DefaultClientFactory creates a real Gemini API client.
func DefaultClientFactory(ctx context.Context, apiKey string) (GenerativeClient, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &genaiClient{inner: c}, nil
}

// GeminiClient implements Client using the Google Gemini API.
type GeminiClient struct {
	apiKey  string
	model   string
	factory ClientFactory
}

// NewGeminiClient creates a new GeminiClient.
// The apiKey must be non-empty; callers should validate before construction.
// The factory creates the underlying generative client; use DefaultClientFactory for production.
func NewGeminiClient(apiKey, model string, factory ClientFactory) *GeminiClient {
	if model == "" {
		model = "gemini-3-pro"
	}
	if factory == nil {
		factory = DefaultClientFactory
	}
	return &GeminiClient{
		apiKey:  apiKey,
		model:   model,
		factory: factory,
	}
}

const (
	maxRetries     = 3
	requestTimeout = 30 * time.Second
	initialBackoff = 1 * time.Second
)

// Review sends a prompt to Gemini and returns structured errors.
// Uses structured output mode for reliable JSON responses.
// Retries up to 3 times with exponential backoff (1s → 2s → 4s).
func (c *GeminiClient) Review(ctx context.Context, prompt string) ([]parser.StructuredError, error) {
	log := logger.FromContext(ctx)
	log.Info("starting LLM review", "model", c.model)
	start := time.Now()

	client, err := c.factory(ctx, c.apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr(float32(0)),
		ResponseMIMEType: "application/json",
		ResponseSchema:   structuredErrorSchema(),
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := range maxRetries {
		log.Debug("LLM request attempt", "attempt", attempt+1, "model", c.model)

		reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		resp, err := client.GenerateContent(reqCtx, c.model, genai.Text(prompt), config)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			log.Warn("LLM request failed, retrying",
				"attempt", attempt+1,
				"error", err,
				"backoff", backoff,
			)

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("LLM review cancelled: %w", ctx.Err())
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}

		// Extract text from response
		text, err := extractText(resp)
		if err != nil {
			return nil, fmt.Errorf("extracting response text: %w", err)
		}

		// Parse structured output
		var result []parser.StructuredError
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("parsing LLM response: %w", err)
		}

		// Set the Tool field on all entries
		for i := range result {
			result[i].Tool = c.model
		}

		duration := time.Since(start)
		log.Info("LLM review complete",
			"model", c.model,
			"issues", len(result),
			"duration_ms", duration.Milliseconds(),
		)

		return result, nil
	}

	return nil, fmt.Errorf("LLM review failed after %d attempts: %w", maxRetries, lastErr)
}

// extractText pulls the text content from a Gemini response.
func extractText(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", errors.New("empty response from Gemini")
	}
	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", errors.New("no content parts in response")
	}
	part := candidate.Content.Parts[0]
	if part.Text == "" {
		return "", errors.New("empty text in response part")
	}
	return part.Text, nil
}

// structuredErrorSchema returns the JSON schema for []StructuredError
// used with Gemini's structured output mode.
func structuredErrorSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"file":     {Type: genai.TypeString, Description: "File path relative to project root"},
				"line":     {Type: genai.TypeInteger, Description: "Line number (1-based)"},
				"severity": {Type: genai.TypeString, Enum: []string{"error", "warning", "info"}},
				"message":  {Type: genai.TypeString, Description: "Issue description"},
				"hint":     {Type: genai.TypeString, Description: "Actionable fix suggestion"},
			},
			Required: []string{"file", "line", "severity", "message"},
		},
	}
}
