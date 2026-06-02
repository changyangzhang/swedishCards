package parser

import (
	"regexp"
	"strings"
	"time"

	"swedishCards/internal/model"
)

type Result struct {
	NoteDate *time.Time
	Entries  []model.ParsedEntry
}

// ParseNotes turns a raw paste of lesson notes into a Result.
// The function is pure and runs offline; Claude enrichment happens later.
func ParseNotes(raw string) Result {
	var out Result

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Date header? If so, set date and skip.
		if d, ok := tryParseDate(line); ok {
			d := d
			out.NoteDate = &d
			continue
		}

		entry, ok := parseEntryLine(line)
		if !ok {
			continue
		}
		out.Entries = append(out.Entries, entry)
	}

	return out
}

func parseEntryLine(line string) (model.ParsedEntry, bool) {
	// Equals-separated line: swedish = english(,english2,...)
	if idx := strings.Index(line, "="); idx >= 0 {
		lhs := strings.TrimSpace(line[:idx])
		rhs := strings.TrimSpace(line[idx+1:])
		if lhs == "" {
			return model.ParsedEntry{}, false
		}

		swedishRaw := lhs
		english := rhs

		kind := classifyKind(lhs)
		swedish := canonicalSwedish(lhs, kind)

		return model.ParsedEntry{
			Kind:       kind,
			Swedish:    swedish,
			SwedishRaw: swedishRaw,
			English:    english,
		}, true
	}

	// Bare line, treat as untranslated sentence (or single untranslated word).
	swedishRaw := line
	swedish := canonicalSwedish(line, model.KindSentenceUntranslated)
	return model.ParsedEntry{
		Kind:       model.KindSentenceUntranslated,
		Swedish:    swedish,
		SwedishRaw: swedishRaw,
		English:    "",
	}, true
}

func classifyKind(lhs string) model.Kind {
	lower := strings.ToLower(lhs)

	// Verb: starts with "att "
	if strings.HasPrefix(lower, "att ") {
		return model.KindVerb
	}

	tokens := strings.Fields(lower)
	if len(tokens) <= 1 {
		return model.KindWord
	}

	// Multi-token: sentence if it contains a pronoun or auxiliary verb,
	// otherwise it's a phrase.
	for _, t := range tokens {
		clean := strings.Trim(t, ".,!?;:")
		if swedishPronouns[clean] || swedishVerbs[clean] {
			return model.KindSentence
		}
	}
	return model.KindPhrase
}

// canonicalSwedish returns the form used for hashing/dedup.
// - lowercased
// - trailing punctuation .!? trimmed
// - "att " prefix stripped for verbs
func canonicalSwedish(s string, kind model.Kind) string {
	out := strings.ToLower(strings.TrimSpace(s))
	out = strings.TrimRight(out, ".!?")
	out = strings.TrimSpace(out)
	if kind == model.KindVerb && strings.HasPrefix(out, "att ") {
		out = strings.TrimSpace(out[4:])
	}
	return out
}

// Date parsing.
//
// Order of attempts:
//  1. DMY short ("2/1/06") — matches "1/6/26" -> 1 June 2026
//  2. ISO ("2006-01-02")
//  3. Long month-name format (e.g. "January 6, 2026")
//
// We deliberately pick DMY over MDY for the short slashed form because
// Swedish-context notes use DMY.

var monthNameRe = regexp.MustCompile(`^(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:,\s*(\d{2,4}))?$`)

func tryParseDate(line string) (time.Time, bool) {
	if t, err := time.Parse("2/1/06", line); err == nil {
		return t, true
	}
	if t, err := time.Parse("2/1/2006", line); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", line); err == nil {
		return t, true
	}
	if m := monthNameRe.FindStringSubmatch(line); m != nil {
		layout := "January 2, 2006"
		if m[3] == "" {
			layout = "January 2"
		}
		if t, err := time.Parse(layout, line); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
