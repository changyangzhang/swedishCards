package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// swedishMarkers are high-frequency Swedish function words that (unlike "i",
// "om", "man", "en", …) are NOT also English words, so seeing one inside an
// "english" value is a reliable sign the model echoed Swedish instead of
// translating. Kept deliberately small and unambiguous.
var swedishMarkers = map[string]bool{
	"och": true, "att": true, "är": true, "jag": true, "det": true,
	"för": true, "på": true, "med": true, "inte": true, "han": true,
	"hon": true, "vi": true, "de": true, "den": true, "ett": true,
	"som": true, "när": true, "vad": true, "hur": true, "så": true,
	"till": true, "eller": true, "har": true, "kan": true, "ska": true,
}

// needsTranslation reports whether an "english" value is missing or is actually
// Swedish. The model occasionally leaves it blank or echoes the Swedish text,
// which would show up as a missing or Swedish "translation" on the card.
func needsTranslation(swedish, english string) bool {
	e := strings.TrimSpace(english)
	if e == "" {
		return true
	}
	if strings.EqualFold(e, strings.TrimSpace(swedish)) {
		return true
	}
	if strings.ContainsAny(e, "åäöÅÄÖ") {
		return true
	}
	for _, w := range strings.Fields(strings.ToLower(e)) {
		if swedishMarkers[strings.Trim(w, ".,!?;:\"'()")] {
			return true
		}
	}
	return false
}

// repairTranslations finds entries and example sentences whose english field is
// missing or Swedish, translates just those in one dedicated call, and patches
// them in place. Best-effort: on any failure the (imperfect) result is left as
// is — the review UI still guards against showing a Swedish hint.
func (c *Client) repairTranslations(ctx context.Context, res *ParseResult) {
	need := map[string]bool{}
	for i := range res.Entries {
		if needsTranslation(res.Entries[i].Swedish, res.Entries[i].English) {
			need[res.Entries[i].Swedish] = true
		}
	}
	for i := range res.ExampleSentences {
		if needsTranslation(res.ExampleSentences[i].Swedish, res.ExampleSentences[i].English) {
			need[res.ExampleSentences[i].Swedish] = true
		}
	}
	if len(need) == 0 {
		return
	}

	texts := make([]string, 0, len(need))
	for s := range need {
		texts = append(texts, s)
	}
	m, err := c.translate(ctx, texts)
	if err != nil || len(m) == 0 {
		return // best-effort
	}

	patch := func(swedish string, english *string) {
		if !needsTranslation(swedish, *english) {
			return
		}
		if v, ok := m[strings.ToLower(strings.TrimSpace(swedish))]; ok && v != "" {
			*english = v
		}
	}
	for i := range res.Entries {
		patch(res.Entries[i].Swedish, &res.Entries[i].English)
	}
	for i := range res.ExampleSentences {
		patch(res.ExampleSentences[i].Swedish, &res.ExampleSentences[i].English)
	}
}

// translate turns a batch of Swedish texts into English via a dedicated,
// single-purpose call — far more reliable than trusting the big combined
// parse+enrich prompt to fill every english field. Returns a map keyed by the
// lower-cased, trimmed Swedish text.
func (c *Client) translate(ctx context.Context, texts []string) (map[string]string, error) {
	payload, err := json.Marshal(texts)
	if err != nil {
		return nil, err
	}

	schema := obj(map[string]any{
		"translations": arr(obj(map[string]any{
			"swedish": strType,
			"english": strType,
		})),
	})

	content, _, err := c.doChat(ctx, chatRequest{
		Model:               c.model,
		MaxCompletionTokens: maxEnrichTokens,
		ReasoningEffort:     reasoningEffort,
		ResponseFormat:      jsonSchemaFormat("translations", schema),
		Messages: []chatMessage{
			{Role: "system", Content: translateSystemPrompt},
			{Role: "user", Content: string(payload)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}

	var out struct {
		Translations []struct {
			Swedish string `json:"swedish"`
			English string `json:"english"`
		} `json:"translations"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, fmt.Errorf("decode translations: %w", err)
	}

	m := make(map[string]string, len(out.Translations))
	for _, t := range out.Translations {
		m[strings.ToLower(strings.TrimSpace(t.Swedish))] = t.English
	}
	return m, nil
}
