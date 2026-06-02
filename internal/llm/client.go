package llm

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/genai"
)

// Client wraps the Gemini SDK with our request shape.
type Client struct {
	sdk   *genai.Client
	model string
}

// Options lets callers (mostly tests) override the underlying HTTP client and
// base URL.
type Options struct {
	HTTPClient *http.Client
	BaseURL    string // empty = default Gemini endpoint
}

// NewClient constructs a Client for the Gemini Developer API.
// Returns an error if apiKey is empty.
func NewClient(ctx context.Context, apiKey, model string, opts Options) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("missing GEMINI_API_KEY")
	}
	cfg := &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: opts.HTTPClient,
	}
	if opts.BaseURL != "" {
		cfg.HTTPOptions = genai.HTTPOptions{BaseURL: opts.BaseURL}
	}
	sdk, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Client{sdk: sdk, model: model}, nil
}
