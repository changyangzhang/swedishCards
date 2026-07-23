# Swedish Cards

A personal, Clozemaster-style web app for learning Swedish. Feed it your raw
lesson notes — or a photo of them — and it turns them into AI-enriched
flashcards you review daily with quizzes that vary every time.

Single-user by design: built for one person's deck, not shared decks or
multi-tenancy.

## How it works

```
  notes / photo ──▶  GPT-5 parse + enrich  ──▶  cards  ──▶  daily review
   (/import)          (translate, examples,        (SQLite)     (/review)
                       cloze hints, typos)                     SM-2 schedule
```

Two things you do: **import** notes to grow the deck, and **review** daily.
Everything else (scheduling, quiz variation, audio, stats) happens around
those two loops.

## Importing notes

On `/import` you can either **paste text** or **upload a file** (image,
PDF, or `.txt`/`.md`, up to 8 MB). A photo of handwritten or printed notes
works directly — GPT-5 does OCR, parsing, and enrichment in one call, so
there's no local OCR step. A spinner shows during the 10–30 s round-trip.

When `OPENAI_API_KEY` is set, GPT-5 mini handles parsing **and** enrichment
together, calibrated for a B1/B2 learner — it skips elementary words and
focuses on:

- Idiomatic phrases and collocations (`ta tag i`, `ge sig av`)
- Less-common vocabulary (`frånkopplad`, `utmattad`)
- Full conversational sentences (`Hur är läget?`)
- Particle verbs and grammar-rich infinitives

For each entry it also fills in an English translation, an example sentence
(used later for cloze prompts), a smart cloze-target word, a one-line grammar
note, and a typo flag when the source looks misspelled.

**Robustness for big imports:**
- Large text pastes are **auto-split** into ~4 KB line-aligned chunks (never
  mid-entry), parsed sequentially, and merged — so a long note can't overflow
  the model's JSON response.
- Low `reasoning_effort` is used: for schema-constrained extraction it keeps
  latency and cost down without hurting quality.
- Retries transient 429/5xx with backoff. If parsing fails outright, the note
  is rolled back so you get a clean retry.

Each entry becomes **exactly one card**, and imports are hash-deduped —
re-pasting the same notes or re-uploading the same file is a no-op.

Without an API key the app still runs: the paste path falls back to an offline
heuristic parser for the classic `Swedish = English` format. File uploads
require the key (they need the multimodal model).

## Reviewing

Every review is a quiz, and **each render picks a fresh presentation** so the
same card never looks identical twice:

| Mode | Prompt | Answer |
|---|---|---|
| `mc_translate` | Swedish | 4 English options |
| `mc_translate_rev` | English | 4 Swedish options |
| `mc_cloze` | Swedish sentence, one word blanked | 4 Swedish word options |

- Distractors are pulled fresh each render (`ORDER BY RANDOM()`).
- For cloze cards the **blanked word rotates** — you might see
  `Jag tränar med ____` once and `Jag ____ med vikter` the next time.
- Word entries with an example sentence use it for the cloze, so you drill
  the word in context.

**Type-in difficulty ramp.** Once you've answered a cloze card correctly twice
in a row (`repetitions >= 2`), `mc_cloze` switches from multiple choice to a
text input — you type the blanked word. A ⌨️ badge marks the harder mode. Any
wrong answer resets the streak and drops it back to MC. Grading is lenient:
case-insensitive, diacritics optional (`tranar` matches `tränar`), trailing
punctuation ignored. Type-in is cloze-only, so the answer is always a single
word — you're never asked to reproduce a whole sentence.

**Auto-graded, no self-rating.** Correct → SM-2 `Good`; wrong → `Again`. The
next card slides in immediately with a green/red ribbon showing the previous
result, and the correct Swedish is **auto-spoken** as it appears.

**Progress bar.** A bar and `N / target` counter above each card track how
far you are through today's session. The count uses distinct cards (relearn
re-attempts don't inflate it), and the denominator grows if you practice
beyond the target.

**Relearn queue.** Cards you get wrong today are re-served at the tail of the
session and keep coming back until you answer them correctly. Re-attempts do
**not** count against your daily target (counting uses distinct cards, not
review rows), so a mistake never eats into your remaining budget.

**Delete mid-session.** A 🗑 button on every card drops it from the deck if it
isn't worth learning; the next card swaps in via HTMX with no reload.

## Scheduling (SM-2)

Standard Anki-style SM-2:

- Default ease factor 2.5, floored at 1.3.
- Intervals: 1 day → 6 days → `prev_interval × ease_factor`.
- `Again` resets repetitions to 0 and bumps lapses.

The **daily review target** (configurable; default 10) covers new and
due-again cards *combined*. The queue serves a ~30% / 70% mix of new vs.
due-again, so you keep learning while maintaining what you know.

When the queue empties, `/review` explains why and offers actions:
- **Practice 5 / 10 more** — bumps the daily target (persisted to SQLite).
- **Review next card anyway** — bypasses the schedule to serve the
  soonest-due card. Reviewing early still updates SM-2 state.

## Audio

🔊 buttons sit next to every Swedish element, using the browser's Web Speech
API (free, on-device) with `lang="sv-SE"` — Chrome/Android uses Google's
neural voice, iOS Safari uses Apple's. After each answer the correct Swedish
auto-plays as the next card appears (the answer click is the gesture that
unlocks autoplay). Cloze placeholders are read as a pause, not "underscore".

## Managing cards

- `/cards` — paginated list; per-row ✎ edit and 🗑 delete, plus bulk-select
  and "Delete N selected". Deleting a card removes its underlying entry so a
  later backfill won't recreate it.
- **Typo suggestions.** AI-flagged corrections appear at the top of `/cards`
  with one-click **Accept** (rewrites the entry, regenerates cards) or
  **Dismiss**.
- `/settings` — adjust the daily review target from the UI; the saved value
  persists and overrides the env seed.
- `/stats` — streak, total reviews, accuracy %, a 30-day reviews histogram,
  and a 14-day due-card forecast, all as inline SVG (no JS chart library).

## Stack

- **Backend:** Go 1.26, `chi` router, `html/template`, `slog`.
- **Storage:** SQLite via `modernc.org/sqlite` (pure Go, no CGO) — one file.
- **Frontend:** server-rendered HTML + [HTMX](https://htmx.org) partial swaps.
  Hand-rolled CSS, vanilla JS for keyboard shortcuts and TTS. No framework.
- **AI:** OpenAI Chat Completions (GPT-5 mini) with strict `json_schema`
  structured output + multimodal image/PDF input. Called over plain `net/http`
  — no SDK dependency.
- **Deploy:** Docker (distroless, ~8 MB image) → Fly.io via `fly.toml`.

## Quick start (local)

```bash
git clone https://github.com/<you>/swedishCards.git
cd swedishCards

cp .env.example .env
# Set OPENAI_API_KEY (create one at https://platform.openai.com/api-keys).
# BASIC_USER/BASIC_PASS are optional locally, required for any public deploy.

docker compose up -d --build
open http://127.0.0.1:8080
```

If port 8080 is taken, set `HTTP_PORT=8765` (or anything free) in `.env` and
re-run `docker compose up -d`.

## Environment variables

| Var | Default | Purpose |
|---|---|---|
| `OPENAI_API_KEY` | _empty_ | Enables AI parsing/enrichment. App works without it (paste-only, heuristic parser); file uploads require it. |
| `OPENAI_MODEL` | `gpt-5-mini` | Model override (e.g. `gpt-5-nano` for lower cost). |
| `NEW_PER_DAY` | `10` | **Seed only** — first-run default for the daily review target (new + due combined). After that `/settings` (DB-persisted) wins. |
| `REVERSE_CARDS` | `false` | Also create English→Swedish cards for word/phrase/verb entries (roughly doubles vocab cards). |
| `BASIC_USER`, `BASIC_PASS` | _empty_ | HTTP Basic Auth. Both empty = disabled (fine for localhost; **set both before exposing publicly**). |
| `DB_PATH` | `swedish.db` | SQLite file path. Compose mounts a volume at `/data`. |
| `HTTP_ADDR` | `:8080` | Listen address. |
| `HTTP_PORT` | `8080` | Host port (compose only). |

## Deploy to Fly.io

```bash
flyctl auth login
flyctl launch --copy-config --no-deploy
flyctl volumes create data --region arn --size 1
flyctl secrets set OPENAI_API_KEY=... BASIC_USER=... BASIC_PASS=...
flyctl deploy
flyctl open
```

`fly.toml` is checked in: single machine, 256 MB, Stockholm region, auto-stop
when idle. Realistic personal-use cost: $0–2/month. Public deploys **require**
`BASIC_USER` + `BASIC_PASS`, or anyone with the URL can use your deck and burn
your OpenAI credit.

## Project layout

```
.
├── main.go                    # thin entrypoint
├── cmd/server/server.go       # wiring: db, router, signal handling
├── internal/
│   ├── cards/                 # entry → card (1:1)
│   ├── config/                # env-var loading
│   ├── llm/                   # OpenAI client, prompts, chunking, schema
│   ├── model/                 # Kind / CardType enums + shared types
│   ├── parser/                # heuristic parser + Swedish stop-words
│   ├── srs/                   # SM-2 math + tests
│   ├── store/                 # SQLite: schema, queries, review queue
│   └── web/
│       ├── auth.go            # HTTP basic auth middleware
│       ├── handlers.go        # HTTP handlers
│       ├── quiz.go            # cloze rotation, choice building, grading
│       ├── router.go          # chi routes
│       ├── static/app.css     # all styling
│       └── templates/*.html   # server-rendered pages
├── Dockerfile                 # multi-stage → distroless
├── docker-compose.yml         # local dev stack
└── fly.toml                   # Fly.io app config
```

## Privacy

- Notes, cards, and review history live in **one local SQLite file**.
- OpenAI is called only at import time. Nothing leaves the machine for
  review, scheduling, or stats.
- Uploaded files are **not stored** — bytes go to OpenAI in the request body,
  then the in-memory copy is garbage-collected. The note row records only
  filename + sha256 + size as a dedup identifier.
- TTS runs entirely in the browser — no audio leaves the device.
- `data/` is gitignored; never commit it.
- OpenAI's API does not train on data sent via the API by default, but review
  their current terms — or paste sensitive notes manually to keep them local.

## License

MIT. See `LICENSE`.
