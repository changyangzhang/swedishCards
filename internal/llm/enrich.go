package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	"swedishCards/internal/model"
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

	temp := float32(0.2)
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    responseSchema(),
		Temperature:       &temp,
		MaxOutputTokens:   8192,
	}

	resp, err := c.sdk.Models.GenerateContent(ctx, c.model,
		[]*genai.Content{genai.NewContentFromText(string(userJSON), genai.RoleUser)},
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	raw := resp.Text()
	if raw == "" {
		return nil, fmt.Errorf("empty response from gemini")
	}

	var result Result
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("decode response: %w (raw=%s)", err, truncate(raw, 500))
	}
	return &result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
