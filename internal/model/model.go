package model

import "time"

type Kind string

const (
	KindWord                 Kind = "word"
	KindPhrase               Kind = "phrase"
	KindVerb                 Kind = "verb"
	KindSentence             Kind = "sentence"
	KindSentenceUntranslated Kind = "sentence_untranslated"
	KindExampleSentence      Kind = "example_sentence"
)

type CardType string

const (
	CardTypeFlashSvEn CardType = "flash_sv_en"
	CardTypeFlashEnSv CardType = "flash_en_sv"
	CardTypeCloze     CardType = "cloze"
)

type ParsedEntry struct {
	Kind       Kind
	Swedish    string // canonical (lowercased, "att " stripped)
	SwedishRaw string // as typed
	English    string // "" if untranslated
}

type Entry struct {
	ID                 int64
	NoteID             int64
	SourceEntryID      *int64
	Kind               Kind
	Swedish            string
	SwedishRaw         string
	English            *string
	SuggestedClozeWord *string
	GrammarNote        *string
	TypoCorrection     *string
	Hash               string
	EnrichedAt         *time.Time
	CreatedAt          time.Time
}

type Card struct {
	ID           int64
	EntryID      int64
	CardType     CardType
	Front        string
	Back         string
	ClozeAnswer  *string
	Hash         string
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	DueAt        time.Time
	LastReviewed *time.Time
	Lapses       int
	CreatedAt    time.Time
}
