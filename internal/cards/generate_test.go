package cards

import (
	"testing"

	"swedishCards/internal/model"
)

func TestGenerate_WordOneCard(t *testing.T) {
	cs := Generate(model.KindWord, "Praktiskt", "convenient, practical", nil)
	if len(cs) != 1 {
		t.Fatalf("got %d cards, want 1", len(cs))
	}
	if cs[0].Front != "Praktiskt" || cs[0].Back != "convenient, practical" {
		t.Errorf("unexpected card: %+v", cs[0])
	}
	if cs[0].ClozeAnswer != nil {
		t.Errorf("word card should not have cloze_answer; got %v", cs[0].ClozeAnswer)
	}
}

func TestGenerate_WordWithoutEnglishNoCard(t *testing.T) {
	cs := Generate(model.KindWord, "Praktiskt", "", nil)
	if len(cs) != 0 {
		t.Errorf("word with no English should produce no card; got %+v", cs)
	}
}

func TestGenerate_SentenceOneCardWithClozeHint(t *testing.T) {
	cs := Generate(model.KindSentence, "Jag gör styrketräning", "I do strength training", nil)
	if len(cs) != 1 {
		t.Fatalf("got %d cards, want 1", len(cs))
	}
	if cs[0].ClozeAnswer == nil || *cs[0].ClozeAnswer != "styrketräning" {
		t.Errorf("cloze answer should default to longest non-stopword; got %v", cs[0].ClozeAnswer)
	}
	if cs[0].Back != "I do strength training" {
		t.Errorf("back wrong: %q", cs[0].Back)
	}
}

func TestGenerate_SentenceUntranslatedOneCardClozeOnly(t *testing.T) {
	cs := Generate(model.KindSentenceUntranslated, "Jag tränar med vikter", "", nil)
	if len(cs) != 1 {
		t.Fatalf("got %d cards, want 1", len(cs))
	}
	if cs[0].Back != "" {
		t.Errorf("untranslated should have empty back; got %q", cs[0].Back)
	}
	if cs[0].ClozeAnswer == nil || *cs[0].ClozeAnswer != "vikter" {
		t.Errorf("cloze answer = %v, want vikter", cs[0].ClozeAnswer)
	}
}

func TestGenerate_ExampleSentenceNoCard(t *testing.T) {
	hint := "praktiskt"
	cs := Generate(model.KindExampleSentence, "Det är praktiskt att bo nära jobbet", "It is practical to live near work", &hint)
	if len(cs) != 0 {
		t.Errorf("example_sentence should produce no card; got %+v", cs)
	}
}

func TestGenerate_PhraseOneCard(t *testing.T) {
	cs := Generate(model.KindPhrase, "Senaste åren", "the last years", nil)
	if len(cs) != 1 || cs[0].Front != "Senaste åren" {
		t.Errorf("phrase should produce 1 card; got %+v", cs)
	}
}

func TestPickClozeAnswer_PrefersHint(t *testing.T) {
	hint := "tränar"
	got := pickClozeAnswer("Jag tränar med vikter", &hint)
	if got != "tränar" {
		t.Errorf("got %q, want hint 'tränar'", got)
	}
}

func TestPickClozeAnswer_HintMissesFallsBack(t *testing.T) {
	hint := "notinthere"
	got := pickClozeAnswer("Jag tränar med vikter", &hint)
	if got != "vikter" {
		t.Errorf("got %q, want fallback 'vikter' (longest non-stopword)", got)
	}
}

func TestPickClozeAnswer_AllStopwordsEmpty(t *testing.T) {
	got := pickClozeAnswer("jag är med", nil)
	if got != "" {
		t.Errorf("got %q, want empty for stopword-only sentence", got)
	}
}
