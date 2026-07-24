package web

import (
	"math/rand/v2"
	"strings"

	"swedishCards/internal/parser"
)

// rotateBlank picks a random non-stopword token from the sentence and returns
// the front (with that token replaced by ____) and the blanked answer.
// Returns (sentence, "") when no eligible token exists.
func rotateBlank(sentence string) (front, answer string) {
	words := strings.Fields(sentence)
	candidates := make([]int, 0, len(words))
	for i, w := range words {
		clean := stripTrailingPuncts(strings.ToLower(w))
		if clean == "" || parser.SwedishStopwords[clean] {
			continue
		}
		candidates = append(candidates, i)
	}
	if len(candidates) == 0 {
		return sentence, ""
	}
	pick := candidates[rand.IntN(len(candidates))]
	masked := make([]string, len(words))
	copy(masked, words)
	answer = stripTrailingPuncts(words[pick])
	masked[pick] = clozeBlankFor(words[pick])
	front = strings.Join(masked, " ")
	return
}

// buildChoices puts the correct answer at a random position among the
// distractors and returns the full shuffled list.
func buildChoices(correct string, distractors []string) []string {
	out := make([]string, 0, 1+len(distractors))
	out = append(out, correct)
	out = append(out, distractors...)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func stripTrailingPuncts(s string) string {
	return strings.TrimRight(s, ".,!?;:\"")
}

// looksSwedish reports whether s appears to be Swedish rather than the English
// translation we expect. Used to suppress a bogus cloze hint when the model
// mistakenly returns Swedish in an "english" field. Swedish uses å/ä/ö, which
// virtually never appear in English text, so their presence is a strong signal.
func looksSwedish(s string) bool {
	return strings.ContainsAny(s, "åäöÅÄÖ")
}

func clozeBlankFor(token string) string {
	body := stripTrailingPuncts(token)
	trail := token[len(body):]
	return "____" + trail
}

// normalizeAnswer collapses answers to a comparable form for type-in grading:
// lower-case, trim whitespace/punctuation, and fold Swedish diacritics
// (å→a, ä→a, ö→o) so phone users don't need the extra keyboard layer to score
// a card correct. Multiple internal spaces collapse to one.
func normalizeAnswer(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, ".,!?;:\"'“”‘’()[]")
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		switch r {
		case 'å', 'ä':
			r = 'a'
		case 'ö':
			r = 'o'
		case 'é', 'è', 'ê':
			r = 'e'
		}
		if r == ' ' || r == '\t' {
			if prevSpace {
				continue
			}
			prevSpace = true
			b.WriteRune(' ')
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

// answersMatch is the grading rule for both MC and type-in review modes:
// lenient case + diacritic-folded comparison. See normalizeAnswer.
func answersMatch(a, b string) bool {
	return normalizeAnswer(a) == normalizeAnswer(b)
}
