package web

import (
	"strings"
	"testing"
)

func TestRotateBlank_PicksNonStopword(t *testing.T) {
	front, answer := rotateBlank("Jag tränar med vikter")
	if answer != "tränar" && answer != "vikter" {
		t.Errorf("answer=%q must be tränar or vikter (jag/med are stopwords)", answer)
	}
	if !strings.Contains(front, "____") {
		t.Errorf("front=%q must contain ____", front)
	}
}

func TestRotateBlank_NeverPicksStopword(t *testing.T) {
	for i := 0; i < 200; i++ {
		_, a := rotateBlank("Jag tränar med vikter")
		lower := strings.ToLower(a)
		if lower == "jag" || lower == "med" {
			t.Fatalf("iteration %d picked stopword: %q", i, a)
		}
	}
}

func TestRotateBlank_BothCandidatesReachable(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		_, a := rotateBlank("Jag tränar med vikter")
		seen[strings.ToLower(a)] = true
	}
	if !seen["tränar"] || !seen["vikter"] {
		t.Errorf("expected both candidates seen, got %v", seen)
	}
}

func TestRotateBlank_StopwordOnlyEmpty(t *testing.T) {
	front, answer := rotateBlank("jag är med")
	if answer != "" {
		t.Errorf("expected empty answer for stopword-only sentence, got %q", answer)
	}
	if front != "jag är med" {
		t.Errorf("front should be unchanged when no candidates; got %q", front)
	}
}

func TestRotateBlank_PreservesTrailingPunct(t *testing.T) {
	for i := 0; i < 30; i++ {
		front, ans := rotateBlank("Jag tränar med vikter.")
		// answer should be a token without the trailing dot
		if strings.HasSuffix(ans, ".") {
			t.Errorf("answer should not include trailing punct: %q", ans)
		}
		// front should still end with .
		if !strings.HasSuffix(front, ".") {
			t.Errorf("trailing punct lost: %q", front)
		}
	}
}

func TestBuildChoices_AlwaysContainsCorrect(t *testing.T) {
	for i := 0; i < 30; i++ {
		out := buildChoices("right", []string{"a", "b", "c"})
		if len(out) != 4 {
			t.Fatalf("len=%d, want 4", len(out))
		}
		found := false
		for _, v := range out {
			if v == "right" {
				found = true
			}
		}
		if !found {
			t.Errorf("correct answer missing: %v", out)
		}
	}
}

func TestBuildChoices_Shuffles(t *testing.T) {
	atZero := 0
	for i := 0; i < 50; i++ {
		out := buildChoices("right", []string{"a", "b", "c"})
		if out[0] == "right" {
			atZero++
		}
	}
	if atZero == 50 || atZero == 0 {
		t.Errorf("shuffle not shuffling: atZero=%d/50", atZero)
	}
}

func TestBuildChoices_NoDistractors(t *testing.T) {
	out := buildChoices("right", nil)
	if len(out) != 1 || out[0] != "right" {
		t.Errorf("expected [right], got %v", out)
	}
}

func TestLooksSwedish(t *testing.T) {
	swedish := []string{
		"Jag tränar med vikter",
		"Hur är läget?",
		"på nätet",
		"förvirrad",
	}
	english := []string{
		"I train with weights",
		"How are things?",
		"on the internet",
		"convenient, practical",
	}
	for _, s := range swedish {
		if !looksSwedish(s) {
			t.Errorf("expected %q to look Swedish", s)
		}
	}
	for _, e := range english {
		if looksSwedish(e) {
			t.Errorf("expected %q to look English", e)
		}
	}
}

func TestAnswersMatch_LenientOnDiacriticsAndCase(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"tränar", "tranar", true},
		{"Vikter", "vikter", true},
		{"Är", "ar", true},
		{"Hörnet", "hornet", true},
		{"åka", "aka", true},
		{"tränar.", "tränar", true},
		{"  tränar  ", "tränar", true},
		{"café", "cafe", true},
		{"tränar", "vikter", false},
		{"", "vikter", false},
	}
	for _, c := range cases {
		if got := answersMatch(c.a, c.b); got != c.want {
			t.Errorf("answersMatch(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
