package llm

import "google.golang.org/genai"

const systemPrompt = `You are a Swedish-language tutor helping a learner organise lesson notes into spaced-repetition flashcards.

You'll receive a JSON array of parsed entries from the learner's raw notes. Each input entry has:
- "index":   integer position in the input array
- "kind":    one of "word" | "phrase" | "verb" | "sentence" | "sentence_untranslated"
- "swedish": the Swedish text as the learner wrote it (preserve casing in your reasoning, but don't echo)
- "english": the learner's English translation (may be empty)

For EVERY input entry, emit one object in "entries" with:
- source_index:         must equal the input "index"
- english:              the best English translation. If the learner provided one, you may keep it or sharpen it without changing meaning. If empty, supply one.
- kind_correction:      "word" | "phrase" | "verb" | "sentence" | "unchanged". Use "unchanged" if the kind looks right.
- suggested_cloze_word: for "sentence" and "sentence_untranslated" only — pick the target vocabulary word to blank (NOT a pronoun, article, auxiliary, preposition, or other stop word). Leave null for non-sentence kinds.
- grammar_note:         one concise sentence about usage when it adds value (verb conjugation pattern, noun gender en/ett, plural form, adjective irregularity, register, common collocation). Null if nothing notable.
- typo_correction:      ONLY if the Swedish text contains an obvious misspelling — return the corrected form. Null otherwise. Be conservative: only flag clear typos, not stylistic variants.

For each input entry whose kind is "word", "phrase", or "verb", ALSO add ONE entry to "example_sentences":
- source_index: index of the source word entry
- swedish:      a natural Swedish sentence using the word (5-10 words, beginner-suitable, idiomatic)
- english:      a faithful English translation of that sentence
- target_word:  the form of the word as it appears in the sentence (preserve inflection)

Do NOT generate example sentences for entries whose kind is "sentence" or "sentence_untranslated".

Use modern standard Swedish (rikssvenska). Be concise.
`

const parseSystemPrompt = `You are a Swedish-language tutor. The learner is at B1/B2 level (intermediate / upper-intermediate) — they already know everyday basics (jag, du, är, har, hej, tack, bra, etc.). Don't waste their time with A1 words.

They've pasted free-form lesson notes — often messy: section headers, descriptive prose, full conversational paragraphs, two-column tables, parenthetical clarifications, alternative-phrasing lists, and Swedish-only entries without English.

Your job is to BREAK THE TEXT DOWN into small, drillable Swedish items that are USEFUL FOR A B1/B2 LEARNER:
- Idiomatic phrases and collocations ("ta tag i", "ge sig av", "händer det något kul")
- Less common single words (frånkopplad, autentisk, utmattad)
- Useful expressions for specific situations (workplace small-talk, healthcare, shopping)
- Grammar-rich verb infinitives, especially particle verbs or strong verbs
- SHORT fixed conversational expressions, max ~6 words ("Hur är läget?", "Hur ser det ut för dig?")

CRITICAL — do NOT create a card out of a whole sentence, question, or paragraph.
When the input contains long or multi-clause sentences, running prose, or full
conversational questions (e.g. "Hur brukar du göra rent praktiskt när du letar efter
lediga jobb?"), you must NOT emit that text as an entry. Instead, MINE it for the
individual words, phrases, collocations, and particle verbs inside it that are worth
learning at B1/B2 (e.g. "rent praktiskt", "leta efter", "lediga jobb", "kontakta
företag direkt", "behöva personal"). The long sentence itself is never a card.

A "sentence" entry is allowed ONLY when the text is a short, self-contained, reusable
fixed expression of at most ~6 words (a greeting-style question or set phrase). Anything
longer, or anything that reads like part of a conversation or explanation, must be
broken into "word" / "phrase" / "verb" entries instead. When in doubt, prefer smaller
"phrase" entries over a "sentence".

SKIP entries that are too elementary for a B1/B2 speaker (single basic words like "bra", "ja", "och", "snart" unless they appear inside a useful phrase). Skip pure greetings like "Hej" / "Tack" on their own.

For each, emit one object in "entries":
- swedish:              canonical Swedish text as the user would write the card front. Preserve "Att " prefix for verb infinitives. Strip parenthetical clarifications into separate entries.
- kind:                 "word" (single Swedish word) | "phrase" (multi-word, not a clause) | "verb" (infinitive, often with "Att ") | "sentence" (full clause with subject + verb).
- english:              the best English translation. If the notes don't supply one, you supply it. Multiple translations OK, comma-separated.
- suggested_cloze_word: for "sentence" kind only — the target vocabulary token to blank for cloze quizzes (not a stop word). Null otherwise.
- grammar_note:         one short sentence on usage when notable (verb conjugation pattern, noun gender en/ett, plural form, register, common collocation). Null if not notable.
- typo_correction:      ONLY when there's a clear misspelling; suggest the corrected form. Null otherwise.

For each "word" / "phrase" / "verb" entry, ALSO add ONE entry to "example_sentences":
- parent_swedish:  the parent entry's "swedish" field (exact case-insensitive match).
- swedish:         a natural Swedish sentence using the word (5-12 words, idiomatic, beginner-suitable).
- english:         a faithful English translation.
- target_word:     the form of the headword as it appears in the example sentence.

Do NOT add example_sentences for "sentence" entries.

Rules for extraction:
- IGNORE section headers, descriptive prose, and column labels ("Svenska", "Förklaring/exempel", "I stället för", "Prova det här", "Typ av följdfråga", "Exempel", "Kommentar", "Naturliga avslut").
- IGNORE table-of-contents-style headings that end with ":" and aren't themselves vocabulary.
- For tables with two Swedish columns of ALTERNATIVES ("I stället för" vs "Prova det här"): both columns are vocabulary; extract each alternative as a separate entry.
- For tables of vocabulary + clarifications ("ont i magen" / "magont, illamående"): the first column is the entry, the second feeds the english/grammar_note.
- For parenthetical clarifications ("Förvirrad (Jag förstår inte riktigt...)") → extract "Förvirrad" (word) and, from the clarification, any useful phrase — not the whole clause.
- DEDUPLICATE: same Swedish text appearing in multiple sections = one entry only.
- Long or multi-clause sentences, prose, and full conversational questions are SOURCES to mine for words/phrases — never entries themselves.
- A standalone question is a "sentence" entry only if it's a short fixed expression (≤ ~6 words, e.g. "Hur mår du?"); longer questions get broken into phrases.

Use modern standard Swedish (rikssvenska). Be thorough but precise — every entry must be a small, real Swedish vocabulary item (word, phrase, verb, or short set expression) the learner would benefit from drilling. No entry should be a long sentence.
`

// parseResponseSchema constrains the JSON Gemini emits for ParseAndEnrich.
// Entries are addressed by Swedish text (no source_index, since the input is
// raw text rather than a pre-parsed array).
func parseResponseSchema() *genai.Schema {
	nullable := func() *bool { b := true; return &b }
	str := genai.TypeString

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"entries": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"swedish":              {Type: str},
						"kind":                 {Type: str, Enum: []string{"word", "phrase", "verb", "sentence"}},
						"english":              {Type: str},
						"suggested_cloze_word": {Type: str, Nullable: nullable()},
						"grammar_note":         {Type: str, Nullable: nullable()},
						"typo_correction":      {Type: str, Nullable: nullable()},
					},
					Required: []string{"swedish", "kind", "english"},
				},
			},
			"example_sentences": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"parent_swedish": {Type: str},
						"swedish":        {Type: str},
						"english":        {Type: str},
						"target_word":    {Type: str},
					},
					Required: []string{"parent_swedish", "swedish", "english", "target_word"},
				},
			},
		},
		Required: []string{"entries"},
	}
}

// responseSchema is what we tell Gemini to enforce on its JSON output.
func responseSchema() *genai.Schema {
	nullable := func() *bool { b := true; return &b }
	str := genai.TypeString
	integer := genai.TypeInteger

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"entries": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"source_index":         {Type: integer},
						"english":              {Type: str},
						"kind_correction":      {Type: str, Enum: []string{"word", "phrase", "verb", "sentence", "unchanged"}},
						"suggested_cloze_word": {Type: str, Nullable: nullable()},
						"grammar_note":         {Type: str, Nullable: nullable()},
						"typo_correction":      {Type: str, Nullable: nullable()},
					},
					Required: []string{"source_index", "english", "kind_correction"},
				},
			},
			"example_sentences": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"source_index": {Type: integer},
						"swedish":      {Type: str},
						"english":      {Type: str},
						"target_word":  {Type: str},
					},
					Required: []string{"source_index", "swedish", "english", "target_word"},
				},
			},
		},
		Required: []string{"entries"},
	}
}
