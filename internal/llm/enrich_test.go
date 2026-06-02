package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swedishCards/internal/model"
)

func TestEnrich_EmptyInputSkipsAPI(t *testing.T) {
	c := &Client{} // no SDK; .Enrich must not touch it on empty input
	res, err := c.Enrich(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res.Entries) != 0 || len(res.ExampleSentences) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}

func TestEnrich_DecodesResponse(t *testing.T) {
	// What we want Enrich() to return after JSON decoding.
	enrichJSON := `{
		"entries": [
			{"source_index": 0, "english": "convenient, practical", "kind_correction": "unchanged", "grammar_note": "adverb form of praktisk"},
			{"source_index": 1, "english": "ingredients", "kind_correction": "unchanged", "typo_correction": "Ingredienser"}
		],
		"example_sentences": [
			{"source_index": 0, "swedish": "Det är praktiskt att ha en plan.", "english": "It is practical to have a plan.", "target_word": "praktiskt"}
		]
	}`

	// Gemini wraps the model output in a candidates[0].content.parts[0].text envelope.
	gemini, _ := json.Marshal(map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"role":  "model",
					"parts": []map[string]any{{"text": enrichJSON}},
				},
				"finishReason": "STOP",
			},
		},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "generateContent") {
			t.Errorf("unexpected request path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, string(gemini))
	}))
	defer srv.Close()

	c, err := NewClient(context.Background(), "test-key", "gemini-2.5-flash", Options{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	res, err := c.Enrich(context.Background(), []model.ParsedEntry{
		{Kind: model.KindWord, SwedishRaw: "Praktiskt", English: ""},
		{Kind: model.KindWord, SwedishRaw: "Ingrendienser", English: "ingredients"},
	})
	if err != nil {
		t.Fatalf("enrich: %v", err)
	}

	if len(res.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(res.Entries))
	}
	if res.Entries[0].English != "convenient, practical" {
		t.Errorf("entry 0 english = %q", res.Entries[0].English)
	}
	if res.Entries[0].GrammarNote == nil || *res.Entries[0].GrammarNote != "adverb form of praktisk" {
		t.Errorf("entry 0 grammar_note = %v", res.Entries[0].GrammarNote)
	}
	if res.Entries[1].TypoCorrection == nil || *res.Entries[1].TypoCorrection != "Ingredienser" {
		t.Errorf("entry 1 typo_correction = %v", res.Entries[1].TypoCorrection)
	}
	if len(res.ExampleSentences) != 1 || res.ExampleSentences[0].TargetWord != "praktiskt" {
		t.Errorf("example_sentences = %+v", res.ExampleSentences)
	}
}

func TestEnrich_MissingKey(t *testing.T) {
	_, err := NewClient(context.Background(), "", "gemini-2.5-flash", Options{})
	if err == nil {
		t.Errorf("expected error for empty API key")
	}
}
