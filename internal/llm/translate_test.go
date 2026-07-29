package llm

import "testing"

func TestNeedsTranslation(t *testing.T) {
	cases := []struct {
		swedish, english string
		want             bool
	}{
		// Good English translations — no repair needed.
		{"lediga jobb", "job vacancies", false},
		{"på nätet", "online, on the internet", false},
		{"Att kontakta", "to contact", false},
		{"hörnet", "the corner", false},

		// Missing.
		{"lediga jobb", "", true},
		{"lediga jobb", "   ", true},

		// Echoed Swedish (identical).
		{"lediga jobb", "lediga jobb", true},

		// Swedish left in the english field.
		{"Han kontaktar mig", "Han kontaktar mig direkt", true}, // marker "han"
		{"jobb", "söker jobb", true},                            // å/ä/ö via ö? no — "söker" has ö
		{"nätet", "på internet", true},                          // marker "på" + å/ä/ö
	}
	for _, c := range cases {
		if got := needsTranslation(c.swedish, c.english); got != c.want {
			t.Errorf("needsTranslation(%q,%q) = %v, want %v", c.swedish, c.english, got, c.want)
		}
	}
}
