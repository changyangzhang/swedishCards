package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegration_ParseAndEnrich_Live hits the real OpenAI API. It's skipped
// unless OPENAI_API_KEY is set, so `go test ./...` stays offline/free. Run it
// with:
//
//	OPENAI_API_KEY=sk-... go test ./internal/llm -run Live -v
//
// Optionally set OPENAI_MODEL (defaults to gpt-5-mini).
func TestIntegration_ParseAndEnrich_Live(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("set OPENAI_API_KEY to run the live integration test")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5-mini"
	}

	c, err := NewClient(context.Background(), key, model, Options{})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// A long conversational sentence — should be MINED for words/phrases,
	// never saved as a single card.
	raw := "Hur brukar du göra rent praktiskt när du letar efter lediga jobb? " +
		"Söker du mest via stora sajter på nätet, eller kontaktar du företag direkt?"

	res, err := c.ParseAndEnrich(ctx, raw)
	if err != nil {
		t.Fatalf("ParseAndEnrich: %v", err)
	}
	if len(res.Entries) == 0 {
		t.Fatalf("expected entries, got none")
	}

	t.Logf("got %d entries, %d example sentences", len(res.Entries), len(res.ExampleSentences))
	for _, e := range res.Entries {
		t.Logf("  [%s] %-28s = %s", e.Kind, e.Swedish, e.English)
		// Guard against the old bug: no entry should be a whole long sentence.
		if len(e.Swedish) > 60 {
			t.Errorf("entry looks like a long sentence (%d chars): %q", len(e.Swedish), e.Swedish)
		}
	}
}
