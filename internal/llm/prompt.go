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
