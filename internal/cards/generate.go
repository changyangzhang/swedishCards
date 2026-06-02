package cards

import (
	"strings"
	"unicode/utf8"

	"swedishCards/internal/model"
	"swedishCards/internal/parser"
)

type Generated struct {
	CardType    model.CardType
	Front       string
	Back        string
	ClozeAnswer *string
}

// Generate returns AT MOST ONE card for the entry. The review layer decides at
// render time whether to present the card as MC translation, MC cloze, fill-in
// translation, or fill-in cloze — so generation no longer pre-baked separate
// flash and cloze cards.
//
// Mapping:
//   - KindWord / KindPhrase / KindVerb: one card (front=swedishRaw, back=english).
//     english must be non-empty.
//   - KindSentence: one card; cloze_answer is the target word for cloze mode
//     (Gemini hint or longest-non-stopword fallback).
//   - KindSentenceUntranslated: one card; back may be empty (Gemini fills later).
//     cloze_answer is the target word for the only viable mode (cloze).
//   - KindExampleSentence: NONE. These entries live as attached "example data"
//     pulled in at review time when reviewing their parent word entry.
func Generate(kind model.Kind, swedishRaw, english string, clozeHint *string) []Generated {
	switch kind {
	case model.KindWord, model.KindPhrase, model.KindVerb:
		if english == "" {
			return nil
		}
		return []Generated{{
			CardType: model.CardTypeFlashSvEn,
			Front:    swedishRaw,
			Back:     english,
		}}

	case model.KindSentence:
		ans := pickClozeAnswer(swedishRaw, clozeHint)
		var ap *string
		if ans != "" {
			ap = &ans
		}
		return []Generated{{
			CardType:    model.CardTypeFlashSvEn,
			Front:       swedishRaw,
			Back:        english,
			ClozeAnswer: ap,
		}}

	case model.KindSentenceUntranslated:
		ans := pickClozeAnswer(swedishRaw, clozeHint)
		if ans == "" {
			return nil // no gradable answer at all
		}
		ap := ans
		return []Generated{{
			CardType:    model.CardTypeFlashSvEn,
			Front:       swedishRaw,
			Back:        english,
			ClozeAnswer: &ap,
		}}

	case model.KindExampleSentence:
		// No standalone card. The parent word's card pulls this sentence in
		// dynamically when its review picks cloze-mode presentation.
		return nil
	}
	return nil
}

// pickClozeAnswer returns the token to use as the cloze target for a sentence.
// Prefers the Gemini-supplied hint when it actually appears in the sentence;
// otherwise falls back to the longest non-stopword token. Returns "" when the
// sentence has no eligible content word at all.
func pickClozeAnswer(sentence string, hint *string) string {
	if hint != nil && *hint != "" {
		target := strings.ToLower(stripTrailingPunct(*hint))
		for _, w := range strings.Fields(sentence) {
			if strings.ToLower(stripTrailingPunct(w)) == target {
				return stripTrailingPunct(w)
			}
		}
	}
	words := strings.Fields(sentence)
	best := ""
	bestLen := 0
	for _, w := range words {
		clean := stripTrailingPunct(strings.ToLower(w))
		if clean == "" || parser.SwedishStopwords[clean] {
			continue
		}
		n := utf8.RuneCountInString(clean)
		if n >= bestLen {
			bestLen = n
			best = stripTrailingPunct(w)
		}
	}
	return best
}

func stripTrailingPunct(s string) string {
	return strings.TrimRight(s, ".,!?;:\"")
}
