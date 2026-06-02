package parser

import (
	"testing"
	"time"

	"swedishCards/internal/model"
)

const sampleInput = `1/6/26

Praktiskt = convenient, practical
Om jag fastnar = if I get stuck
Att fastna = to get stuck
Att undvika = to avoid
Frånkopplad = disconnected, detached
Senaste åren = the last years
Autentiskt = authentic
Ingrendienser = ingredients
Tyngdlyftning = weightlifting
Jag tränar med vikter
Jag gör styrketräning = I do strengthtraining`

func TestParseNotes_Sample(t *testing.T) {
	r := ParseNotes(sampleInput)

	if r.NoteDate == nil {
		t.Fatalf("expected a note date, got nil")
	}
	want := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if !r.NoteDate.Equal(want) {
		t.Errorf("date mismatch: got %v, want %v", r.NoteDate, want)
	}

	if len(r.Entries) != 11 {
		t.Fatalf("expected 11 entries, got %d", len(r.Entries))
	}

	expected := []struct {
		Kind    model.Kind
		Swedish string
		English string
	}{
		{model.KindWord, "praktiskt", "convenient, practical"},
		{model.KindSentence, "om jag fastnar", "if I get stuck"},
		{model.KindVerb, "fastna", "to get stuck"},
		{model.KindVerb, "undvika", "to avoid"},
		{model.KindWord, "frånkopplad", "disconnected, detached"},
		{model.KindPhrase, "senaste åren", "the last years"},
		{model.KindWord, "autentiskt", "authentic"},
		{model.KindWord, "ingrendienser", "ingredients"},
		{model.KindWord, "tyngdlyftning", "weightlifting"},
		{model.KindSentenceUntranslated, "jag tränar med vikter", ""},
		{model.KindSentence, "jag gör styrketräning", "I do strengthtraining"},
	}

	for i, want := range expected {
		got := r.Entries[i]
		if got.Kind != want.Kind {
			t.Errorf("entry %d: kind = %q, want %q (swedish=%q)", i, got.Kind, want.Kind, got.Swedish)
		}
		if got.Swedish != want.Swedish {
			t.Errorf("entry %d: swedish = %q, want %q", i, got.Swedish, want.Swedish)
		}
		if got.English != want.English {
			t.Errorf("entry %d: english = %q, want %q", i, got.English, want.English)
		}
	}
}

func TestParseNotes_VerbCanonicalStripsAtt(t *testing.T) {
	r := ParseNotes("Att springa = to run")
	if len(r.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(r.Entries))
	}
	e := r.Entries[0]
	if e.Kind != model.KindVerb {
		t.Errorf("kind = %q, want verb", e.Kind)
	}
	if e.Swedish != "springa" {
		t.Errorf("swedish = %q, want %q", e.Swedish, "springa")
	}
	if e.SwedishRaw != "Att springa" {
		t.Errorf("swedishRaw = %q, want %q", e.SwedishRaw, "Att springa")
	}
}

func TestParseNotes_DMYDateInterpretation(t *testing.T) {
	r := ParseNotes("1/6/26\nHej = hello")
	if r.NoteDate == nil {
		t.Fatalf("no date parsed")
	}
	if r.NoteDate.Month() != time.June || r.NoteDate.Day() != 1 {
		t.Errorf("date = %v, want 1 June (DMY)", r.NoteDate)
	}
}

func TestParseNotes_IgnoresBlankLines(t *testing.T) {
	r := ParseNotes("\n\n\nHej = hello\n\n")
	if len(r.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(r.Entries))
	}
}

func TestParseNotes_TrailingPunctuationInHashing(t *testing.T) {
	r := ParseNotes("Hej! = hello")
	if len(r.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(r.Entries))
	}
	if r.Entries[0].Swedish != "hej" {
		t.Errorf("canonical = %q, want %q", r.Entries[0].Swedish, "hej")
	}
}

func TestClassifyKind_PronounMakesSentence(t *testing.T) {
	if k := classifyKind("Jag tränar"); k != model.KindSentence {
		t.Errorf("kind = %q, want sentence", k)
	}
	if k := classifyKind("Senaste åren"); k != model.KindPhrase {
		t.Errorf("kind = %q, want phrase", k)
	}
}
