# Swedish Cards

A personal, Clozemaster-style Swedish-learning web app. Paste raw lesson notes,
get AI-enriched flashcards back, and review them daily with multiple-choice
quizzes that vary every time you see a card.

Single-user app. Built for one person to learn Swedish; not designed for shared
decks or multi-tenancy.

## Features

### Notes → cards pipeline
Two input modes on `/import`:

1. **Paste text** into the textarea. Works for the classic `Swedish = English`
   format and the heuristic parser handles it offline (no API key needed).
2. **Upload a file** — image (PNG/JPG/HEIC/WebP), PDF, or .txt/.md. Up to 8 MB.
   Photos of handwritten or printed Swedish notes work directly; Gemini's
   multimodal API does OCR + parsing in a single call. No local OCR library.

A loading spinner shows on submit so you know the 10–30 s AI round-trip is in
flight; the button disables to prevent double-submits.

Each entry produces **exactly one card** (no duplicate cards per concept).
Hash-based dedup: re-pasting the same notes or re-uploading the same file is
a no-op.

### AI parsing + enrichment (Gemini 2.5 Flash, free tier)
When `GEMINI_API_KEY` is set, Gemini handles BOTH parsing and enrichment in
one call, calibrated for a B1/B2 learner — skips elementary words, focuses on:
- Idiomatic phrases and collocations (`ta tag i`, `ge sig av`)
- Less-common vocabulary (`frånkopplad`, `utmattad`)
- Full conversational sentences (`Hur är läget?`, `Hur ser det ut för dig?`)
- Particle verbs and other grammar-rich infinitives

For each entry Gemini also supplies:
- English translation
- One example sentence per word entry (used later for cloze prompts)
- Smart cloze target words (which word in a sentence to blank)
- One-sentence grammar notes (verb conjugation patterns, noun gender, etc.)
- Typo flags ("Ingrendienser → Ingredienser")

Handles free-form notes the heuristic parser can't: section headers, tables,
parenthetical clarifications, alternative-phrasing lists, bare Swedish
without translations.

Uses the free Gemini API tier — no credit card needed. Retries on transient
503/429 with backoff (2s, 5s, 8s). If parsing fails entirely the note row is
rolled back so you can retry once Gemini is back.

App degrades gracefully when `GEMINI_API_KEY` is unset: the textarea path
still works via the heuristic parser; file uploads require the key.

### Multiple-choice review
Every review is multiple-choice. Each render of a card picks **a fresh
presentation at random**:

| Mode | Prompt | Choices |
|---|---|---|
| `mc_translate` | Swedish | 4 English options |
| `mc_translate_rev` | English | 4 Swedish options |
| `mc_cloze` | Swedish sentence with one word blanked | 4 Swedish word options |

Reviewing the same card twice in a row never looks identical:
- Distractors are pulled fresh each render (`ORDER BY RANDOM()`).
- For cloze cards, the blanked word rotates — you might see
  `Jag tränar med ____` once and `Jag ____ med vikter` next time.
- For word entries that have a Gemini-generated example sentence attached,
  the cloze mode uses that example so you drill the word in context.

Auto-graded: correct → SM-2 `Good`; wrong → `Again`. No self-rating. A small
ribbon at the top shows the previous answer's result before each new card.

### Text-to-speech
🔊 buttons next to every Swedish text element. Uses the browser's Web Speech
API (free, runs on-device) with `lang="sv-SE"`. Chrome on Android uses
Google's neural Swedish voice; iOS Safari uses Apple's. Cloze placeholders
are read as a pause, not "underscore".

### Spaced repetition (SM-2)
Standard Anki-style SM-2:
- Default ease factor 2.5; floored at 1.3.
- Intervals: 1 day → 6 days → `prev_interval × ease_factor`.
- `Again` resets `repetitions` to 0 and bumps `lapses`.
- **Daily review target** (configurable; default 10) covers BOTH new and
  due-again cards combined. The queue serves a ~30%/70% mix of new vs
  due-again, so you keep learning new material while maintaining what you
  already know.

### Daily-cap UX
The `/review` empty state explains *why* it's empty (target reached, all
caught up, etc.) and offers concrete actions:
- **"Practice 5 more"** / **"Practice 10 more"** — bump the daily target
  permanently (persisted to SQLite, takes effect immediately).
- **"Review next card anyway"** — bypass the SM-2 schedule entirely and
  serve the soonest-due card, regardless of its actual due time. SRS state
  still updates from the early review.

### Card management
- `/cards` — paginated list. Filter by kind, due status, has-typo.
- Per-row 🗑 to delete a single card (also deletes the underlying entry, so
  the next backfill doesn't recreate it).
- Bulk-select checkboxes + "Delete N selected" button.
- ✎ to edit a card's front/back/cloze answer.
- AI-suggested typo corrections appear at the top of `/cards` with
  one-click **Accept** (rewrites the entry, regenerates cards) or
  **Dismiss**.
- **🗑 Delete this card** button on every `/review` card — drop a card
  mid-session if it isn't worth learning; the next card slides in via
  HTMX swap (no full reload).

### Stats
`/stats` shows streak, total reviews, accuracy %, a 30-day reviews
histogram, and a 14-day due-card forecast. All charts are inline SVG —
no JS chart library.

### Settings (`/settings`)
Adjust the daily review target from the UI without touching `.env`. Saved
value persists across restarts and overrides the env-var seed.

## Stack

- **Backend**: Go 1.26, `chi` router, `html/template`, `slog`.
- **Storage**: SQLite via `modernc.org/sqlite` (pure Go, no CGO). Single
  file at `/data/swedish.db`.
- **Frontend**: server-rendered HTML + [HTMX](https://htmx.org) for partial
  swaps. Hand-rolled CSS (no Tailwind, no framework). Vanilla JS for
  keyboard shortcuts and TTS.
- **AI**: `google.golang.org/genai` (official Gen AI Go SDK), `responseSchema`
  for structured output.
- **Deploy**: Docker (distroless base, ~5MB binary). Ships to Fly.io with
  the included `fly.toml`.

## Quick start (local)

```bash
git clone https://github.com/<you>/swedishCards.git
cd swedishCards

cp .env.example .env
# Edit .env — at minimum, set GEMINI_API_KEY (free at
# https://aistudio.google.com/apikey). BASIC_USER/BASIC_PASS optional
# for local; required for any public deployment.

docker compose up -d --build
open http://127.0.0.1:8080
```

If port 8080 is taken on your machine, set `HTTP_PORT=8765` (or anything
free) in `.env` and re-run `docker compose up -d`.

## Environment variables

| Var | Default | Purpose |
|---|---|---|
| `GEMINI_API_KEY` | _empty_ | Enables AI enrichment. Free tier at https://aistudio.google.com/apikey. App works without it. |
| `GEMINI_MODEL` | `gemini-2.5-flash` | Override the model. |
| `NEW_PER_DAY` | `10` | **Seed only** — first-run default for the daily review target (new + due combined). After that, the value in `/settings` (DB-persisted) wins. |
| `BASIC_USER`, `BASIC_PASS` | _empty_ | HTTP Basic Auth. Both empty = auth disabled (fine for localhost; **set both before exposing the app publicly**). |
| `DB_PATH` | `swedish.db` | SQLite file path. Docker compose mounts a host volume at `/data`. |
| `HTTP_ADDR` | `:8080` | Listen address. |
| `HTTP_PORT` | `8080` | Host port (compose only). |

## Deploy to Fly.io

```bash
flyctl auth login
flyctl launch --copy-config --no-deploy
flyctl volumes create data --region arn --size 1
flyctl secrets set GEMINI_API_KEY=... BASIC_USER=... BASIC_PASS=...
flyctl deploy
flyctl open
```

`fly.toml` is checked in: single machine, 256 MB, Stockholm region,
auto-stop when idle. Realistic cost for personal use: $0–2/month.

Public deployment requires `BASIC_USER` + `BASIC_PASS` to be set, otherwise
anyone with the URL can use your deck and burn your Gemini quota.

## Project layout

```
.
├── main.go                          # thin entrypoint
├── cmd/server/server.go             # wiring: db, router, signal handling
├── internal/
│   ├── cards/generate.go            # entry → card (1:1 under the current model)
│   ├── config/config.go             # env-var loading
│   ├── llm/                         # Gemini client + structured-output schema
│   ├── model/model.go               # Kind, CardType enums + shared types
│   ├── parser/                      # heuristic parser + Swedish stop-words
│   ├── srs/sm2.go                   # SM-2 math + tests
│   ├── store/                       # SQLite + embedded schema + queries
│   └── web/
│       ├── auth.go                  # HTTP basic auth middleware
│       ├── handlers.go              # all HTTP handlers
│       ├── quiz.go                  # rotateBlank, buildChoices (pure)
│       ├── render.go                # template loading
│       ├── router.go                # chi routes
│       ├── static/app.css           # all styling
│       └── templates/*.html         # server-rendered pages
├── Dockerfile                       # multi-stage → distroless
├── docker-compose.yml               # local dev stack
└── fly.toml                         # Fly.io app config
```

## Privacy

- Lesson notes, cards, and review history live in **one local SQLite file**.
- Gemini API is called only at import time (one batched call per new note).
  No data leaves the machine for review / SM-2 / stats.
- File uploads (images/PDFs) are NOT stored — the bytes go to Gemini in the
  request body, then the in-memory copy is garbage-collected. `notes.raw_text`
  records only the filename + sha256 + size as a stable dedup identifier.
- TTS runs entirely in the browser (Web Speech API) — no audio leaves the
  device.
- The `data/` directory is gitignored; never commit it.
- Gemini's free tier may use inputs to improve Google's models per their
  terms; upgrade to paid to opt out, or paste sensitive notes manually.

## License

MIT. See `LICENSE`.
