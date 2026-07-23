package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client talks to the OpenAI Chat Completions API over plain HTTP (no SDK
// dependency). It's used for parsing + enriching Swedish lesson notes with
// structured (JSON-schema) output and multimodal (image/PDF) input.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Options lets callers (mostly tests) override the HTTP client and base URL.
type Options struct {
	HTTPClient *http.Client
	BaseURL    string // empty = api.openai.com
}

// NewClient constructs a Client. Returns an error if apiKey is empty. The ctx
// is accepted for signature symmetry with the previous SDK-based client; no
// network call happens here.
func NewClient(_ context.Context, apiKey, model string, opts Options) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("missing OPENAI_API_KEY")
	}
	c := &Client{
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
	if opts.BaseURL != "" {
		c.baseURL = opts.BaseURL
	}
	if opts.HTTPClient != nil {
		c.httpClient = opts.HTTPClient
	}
	return c, nil
}

// --- request / response wire types ---

// chatMessage.Content is either a plain string or a []contentPart (multimodal).
type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
	File     *filePart `json:"file,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type filePart struct {
	Filename string `json:"filename,omitempty"`
	FileData string `json:"file_data,omitempty"`
}

type chatRequest struct {
	Model               string          `json:"model"`
	Messages            []chatMessage   `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
	ResponseFormat      *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

// jsonSchemaFormat wraps a schema in the strict json_schema response format.
func jsonSchemaFormat(name string, schema map[string]any) *responseFormat {
	return &responseFormat{
		Type:       "json_schema",
		JSONSchema: &jsonSchema{Name: name, Strict: true, Schema: schema},
	}
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// doChat posts a chat-completion request, retrying transient 429/5xx errors
// with linear backoff. Returns the assistant message content + finish reason.
func (c *Client) doChat(ctx context.Context, req chatRequest) (content, finishReason string, err error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		content, finishReason, err = c.doChatOnce(ctx, body)
		if err == nil {
			return content, finishReason, nil
		}
		if !isTransientError(err) {
			return "", "", err
		}
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-time.After(time.Duration(2+3*attempt) * time.Second):
		}
	}
	return "", "", err
}

func (c *Client) doChatOnce(ctx context.Context, body []byte) (string, string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", &httpError{status: resp.StatusCode, body: truncate(string(raw), 300)}
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", "", fmt.Errorf("decode response: %w (raw=%s)", err, truncate(string(raw), 300))
	}
	if cr.Error != nil {
		return "", "", fmt.Errorf("openai error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", "", errors.New("openai: no choices in response")
	}
	return cr.Choices[0].Message.Content, cr.Choices[0].FinishReason, nil
}

type httpError struct {
	status int
	body   string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("openai HTTP %d: %s", e.status, e.body)
}

// isTransientError reports whether the error is worth retrying — HTTP 429
// (rate limit) or any 5xx (server/overload). 4xx (bad key, bad request,
// unsupported location) won't change between attempts, so we don't retry them.
func isTransientError(err error) bool {
	var he *httpError
	if errors.As(err, &he) {
		return he.status == http.StatusTooManyRequests || he.status >= 500
	}
	return false
}
