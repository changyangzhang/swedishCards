package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"swedishCards/internal/model"
)

// Shared generation settings. reasoning_effort "low" suits schema-constrained
// extraction (fast + cheap); the token caps are sized so long imports don't
// truncate mid-JSON (large pastes are also chunked — see splitForImport).
const (
	reasoningEffort = "low"
	maxParseTokens  = 32768
	maxEnrichTokens = 16384
)

// EnrichedEntry is one row in the Gemini response's "entries" array.
type EnrichedEntry struct {
	SourceIndex        int     `json:"source_index"`
	English            string  `json:"english"`
	KindCorrection     string  `json:"kind_correction"`
	SuggestedClozeWord *string `json:"suggested_cloze_word"`
	GrammarNote        *string `json:"grammar_note"`
	TypoCorrection     *string `json:"typo_correction"`
}

// ExampleSentence is one row in the "example_sentences" array.
type ExampleSentence struct {
	SourceIndex int    `json:"source_index"`
	Swedish     string `json:"swedish"`
	English     string `json:"english"`
	TargetWord  string `json:"target_word"`
}

// Result is the decoded Gemini response.
type Result struct {
	Entries          []EnrichedEntry   `json:"entries"`
	ExampleSentences []ExampleSentence `json:"example_sentences"`
}

// promptEntry is the JSON shape we send to Gemini.
type promptEntry struct {
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	Swedish string `json:"swedish"`
	English string `json:"english,omitempty"`
}

// Enrich sends entries to Gemini and returns the structured result. Empty input
// short-circuits without any API call.
func (c *Client) Enrich(ctx context.Context, entries []model.ParsedEntry) (*Result, error) {
	if len(entries) == 0 {
		return &Result{}, nil
	}

	payload := make([]promptEntry, len(entries))
	for i, e := range entries {
		payload[i] = promptEntry{
			Index:   i,
			Kind:    string(e.Kind),
			Swedish: e.SwedishRaw,
			English: e.English,
		}
	}
	userJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	content, finish, err := c.doChat(ctx, chatRequest{
		Model:               c.model,
		MaxCompletionTokens: maxEnrichTokens,
		ReasoningEffort:     reasoningEffort,
		ResponseFormat:      jsonSchemaFormat("enrich_result", responseSchema()),
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: string(userJSON)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai enrich: %w", err)
	}
	if finish == "length" {
		return nil, fmt.Errorf("response truncated (hit max_completion_tokens)")
	}
	if content == "" {
		return nil, fmt.Errorf("empty response from openai")
	}

	var result Result
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (raw=%s)", err, truncate(content, 500))
	}
	return &result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ParsedAndEnrichedEntry is one row in ParseResult.Entries — Gemini has both
// parsed it from raw notes AND filled in translation/grammar/typo data.
type ParsedAndEnrichedEntry struct {
	Swedish            string  `json:"swedish"`
	Kind               string  `json:"kind"`
	English            string  `json:"english"`
	SuggestedClozeWord *string `json:"suggested_cloze_word"`
	GrammarNote        *string `json:"grammar_note"`
	TypoCorrection     *string `json:"typo_correction"`
}

// ParseExampleSentence references its parent by Swedish text rather than an
// array index, since the input to ParseAndEnrich is raw text not an indexed
// array.
type ParseExampleSentence struct {
	ParentSwedish string `json:"parent_swedish"`
	Swedish       string `json:"swedish"`
	English       string `json:"english"`
	TargetWord    string `json:"target_word"`
}

// ParseResult is the decoded Gemini response for ParseAndEnrich.
type ParseResult struct {
	Entries          []ParsedAndEnrichedEntry `json:"entries"`
	ExampleSentences []ParseExampleSentence   `json:"example_sentences"`
}

// ParseAndEnrich asks Gemini to BOTH parse free-form Swedish lesson notes AND
// enrich them in a single call. Handles complex inputs that the heuristic
// parser can't: section headers, tables, parenthetical context, bare Swedish
// phrases without "=" separators. Retries on transient errors (503/429).
func (c *Client) ParseAndEnrich(ctx context.Context, rawText string) (*ParseResult, error) {
	if strings.TrimSpace(rawText) == "" {
		return &ParseResult{}, nil
	}
	chunks := splitForImport(rawText, importChunkChars)
	if len(chunks) == 1 {
		return c.parseContent(ctx, textUserMessage(rawText))
	}

	// Large paste: parse each chunk in its own call and merge. Sequential so
	// we stay within the per-minute request budget; parseContent already
	// retries transient 429/5xx. If any chunk fails, the whole import fails
	// (the caller rolls back the note) so the user gets a clean retry.
	merged := &ParseResult{}
	for i, chunk := range chunks {
		res, err := c.parseContent(ctx, textUserMessage(chunk))
		if err != nil {
			return nil, fmt.Errorf("chunk %d/%d: %w", i+1, len(chunks), err)
		}
		merged.Entries = append(merged.Entries, res.Entries...)
		merged.ExampleSentences = append(merged.ExampleSentences, res.ExampleSentences...)
	}
	return merged, nil
}

// importChunkChars is the target max size of each parse chunk for large text
// imports. Sized well under the model's output budget so even a chunk that's
// all short vocab lines can't overflow the JSON response.
const importChunkChars = 4000

// splitForImport breaks rawText into line-aligned chunks of at most maxChars.
// Splitting only ever happens on newline boundaries so a vocab entry is never
// cut in half. Returns a single chunk when the text already fits.
func splitForImport(rawText string, maxChars int) []string {
	if len(rawText) <= maxChars {
		return []string{rawText}
	}
	var chunks []string
	var b strings.Builder
	for _, line := range strings.SplitAfter(rawText, "\n") {
		// A single monster line longer than maxChars gets its own chunk rather
		// than being split mid-content — the model handles it, and it keeps the
		// entry intact.
		if b.Len() > 0 && b.Len()+len(line) > maxChars {
			chunks = append(chunks, b.String())
			b.Reset()
		}
		b.WriteString(line)
	}
	if b.Len() > 0 {
		chunks = append(chunks, b.String())
	}
	return chunks
}

// ParseAndEnrichFile extracts text from an uploaded image or PDF using Gemini's
// multimodal input, then parses + enriches in the same call. Saves the
// round-trip of a separate OCR step.
//
// Supported mime types: image/png, image/jpeg, image/webp, image/heic,
// image/heif, application/pdf. For text/plain, callers should hand the bytes
// to ParseAndEnrich(string(bytes)) instead — that's strictly cheaper.
func (c *Client) ParseAndEnrichFile(ctx context.Context, data []byte, mimeType string) (*ParseResult, error) {
	if len(data) == 0 {
		return &ParseResult{}, nil
	}
	dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
	instruction := contentPart{
		Type: "text",
		Text: "These are the learner's Swedish lesson notes (image or PDF). Read the content carefully and extract Swedish vocabulary items per the system instructions.",
	}
	// PDFs go through the "file" content part; images through "image_url".
	var media contentPart
	if mimeType == "application/pdf" {
		media = contentPart{Type: "file", File: &filePart{Filename: "notes.pdf", FileData: dataURI}}
	} else {
		media = contentPart{Type: "image_url", ImageURL: &imageURL{URL: dataURI}}
	}
	return c.parseContent(ctx, []chatMessage{
		{Role: "user", Content: []contentPart{instruction, media}},
	})
}

// textUserMessage wraps plain text as a single user message.
func textUserMessage(text string) []chatMessage {
	return []chatMessage{{Role: "user", Content: text}}
}

// parseContent runs the parse-and-enrich call against arbitrary user messages
// (text-only or multimodal). Shared by ParseAndEnrich and ParseAndEnrichFile.
func (c *Client) parseContent(ctx context.Context, userMessages []chatMessage) (*ParseResult, error) {
	messages := append([]chatMessage{{Role: "system", Content: parseSystemPrompt}}, userMessages...)

	content, finish, err := c.doChat(ctx, chatRequest{
		Model:               c.model,
		MaxCompletionTokens: maxParseTokens,
		ReasoningEffort:     reasoningEffort,
		ResponseFormat:      jsonSchemaFormat("parse_result", parseResponseSchema()),
		Messages:            messages,
	})
	if err != nil {
		return nil, fmt.Errorf("openai parse: %w", err)
	}
	if finish == "length" {
		return nil, fmt.Errorf("response truncated (hit max_completion_tokens) — split this note into smaller chunks and import again")
	}
	if content == "" {
		return nil, fmt.Errorf("empty response from openai parse")
	}

	var result ParseResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("decode parse response: %w (raw=%s)", err, truncate(content, 500))
	}
	return &result, nil
}
