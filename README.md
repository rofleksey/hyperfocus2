<p align="center">
  <img src="screenshot.png" alt="Hyperfocus screenshot" width="720">
</p>

<h3 align="center"><a href="https://hyperfocusdbd.com">🔗 hyperfocusdbd.com</a></h3>

# Hyperfocus — Dead by Daylight streamer detection & stream history

Hyperfocus watches every live [Dead by Daylight](https://store.steampowered.com/app/381210/Dead_by_Daylight/) stream on Twitch and tells you — often mid-match — when you're playing against a streamer. Subscribe with your Twitch login and Steam profile, verify in your own chat, and the bot messages you there whenever your Steam name is spotted in another streamer's lobby.

It's also a browsable history of the DBD category: who was live at any moment, with viewer counts, titles, thumbnails, and OCR-read survivor names.

> **Only Dead by Daylight is supported** — the name reader is built specifically for the DBD HUD.

## Features

- **Streamer detection notifications** — optional Twitch chat bot that matches your Steam name against the survivors in other streamers' lobbies and pings you on a hit (see [Notifications](#notifications))
- **Live polling** — continuously fetches all DBD streams from Twitch (~2000 per cycle) in a back-to-back poll loop
- **Historical snapshots** — navigate back in time with prev/next buttons, a datetime picker, or a "Now" button with new-data indicator
- **Thumbnail gallery** — responsive grid of 16:9 stream previews with viewer counts, relevance scores, and infinite scroll
- **OCR survivor names** — extracts the 4 player names from the bottom-left HUD of each stream's 1080p preview (720p fallback) using [PaddleOCR on ONNX Runtime](https://github.com/rofleksey/hyperfocus2-ocr)
- **Fuzzy search** — search by streamer name, language, or survivor in-game name with a loose matcher that tolerates OCR recognition errors
- **Streamer detail** — click any streamer to see their sample at that moment: preview, viewer count, title, tags, and OCR survivor names
- **Stats charts** — line charts for streams online, total viewers, cycle time, disk usage, preview capture rate, and OCR success rate over time
- **No accounts, no cookies** — built with Vue 3 + PrimeVue, dark theme, responsive

## Notifications

The site has two parts: a **landing page** (`/`) that explains the service, and the **live gallery** (`/live`) for browsing the stream history.

### How it works

1. **Subscribe** — enter your Twitch username and your Steam profile URL on `/subscribe`.
2. **Verify** — the bot joins your Twitch chat; type `!hyperfocussub` in your *own* channel to confirm ownership.
3. **Tracking** — every live DBD stream's preview is OCR-read to extract the four survivor names.
4. **Detected** — when your Steam name fuzzy-matches (score ≥ `notify.min_score`, default 0.60) a survivor in another streamer's lobby, the bot pings you in your own Twitch chat with their channel name.

Notifications arrive as **chat messages in your own channel** — the bot does not send direct messages (whispers). Unverified subscriptions expire after 24 hours; unsubscribe anytime with `!hyperfocusunsub` in your chat. Your Steam display name is refreshed automatically.

### Will it detect you?

Detection relies on reading names from stream previews, so it won't catch everything:

- **Anonymous mode** — your nickname is replaced with the character's name, so there is nothing to match
- **Survivor names hidden** — the streamer disabled survivor nicknames in their game settings
- **Obscured HUD** — names covered by overlays, unusual HUD placement, or unreadable previews
- **Very short matches** — roughly under 5 minutes can end before the tracker checks that stream
- **Tricky nicknames** — very short, very common (`cat`, `orange`, `111`), or non-Latin names may be missed or matched to the wrong player

Matching is fuzzy to tolerate OCR errors, so occasional false positives are possible. A per-streamer cooldown (`notify.cooldown`, default 30m) suppresses repeat pings.

## Architecture

```
Twitch Helix API       Steam API
       │                    │
       ▼                    ▼
┌────────────┐    ┌──────────────┐    ┌─────────┐
│   Poller   │───▶│  PostgreSQL  │◀───│  HTTP   │
│ (perpetual)│    │              │    │   API   │
└─────┬──────┘    └──────┬───────┘    └────┬────┘
      │                  │                 │
      ▼                  │                 ▼
┌────────────┐           │         ┌──────────────┐
│  OCR micro │           │         │  Vue 3 SPA   │
│  (ONNX)    │───────────┘         └──────────────┘
└────────────┘
       ▲
┌──────┴──────┐
│  Notify bot │───▶ Twitch IRC
└─────────────┘
```

Each poll cycle: fetch live streams → download preview images → OCR survivor names → store snapshot + samples in Postgres. The HTTP API serves the SPA and a read-only JSON API. Background loops handle retention pruning, optional Steam name resolution, and IRC-based lobby notifications.

## Tech stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, `net/http` (Go 1.22+ routing) |
| Database | PostgreSQL 18, `pgx/v5` |
| Frontend | Vue 3, TypeScript, PrimeVue 4, Chart.js |
| OCR | [hyperfocus2-ocr](https://github.com/rofleksey/hyperfocus2-ocr) — Python, PaddleOCR PP-OCRv3, ONNX Runtime |
| IRC | `go-twitch-irc/v4` |
| Twitch API | `nicklaw5/helix/v2` |
| Infra | Docker Compose, GitHub Actions, Traefik |

## Run locally

```bash
cp config.example.yaml config.yaml   # edit with your Twitch API credentials
go build -o hyperfocus ./cmd/server/
./hyperfocus
```

The embedded Vue SPA is served on `:8080`. Migrations run automatically.

## Config

Preview sizes are fixed: full previews are fetched as 1920×1080 with a
1280×720 fallback, gallery thumbnails as 480×270. All other knobs live in
YAML:

```yaml
service:
  http_addr: ":8080"

db:
  host: "localhost"
  port: 5432
  user: "postgres"
  database: "hyperfocus"

twitch:
  client_id: "..."       # from dev.twitch.tv
  client_secret: "..."

twitch_bot:
  client_id: "..."       # bot app credentials (falls back to twitch.*)
  client_secret: "..."
  refresh_token: "..."   # bot account token (chat:read chat:edit scope)

ocr:
  enabled: true
  api_url: "http://localhost:8081"   # hyperfocus2-ocr service

notify:
  enabled: false                     # optional chat notifications
  min_score: 0.60                    # fuzzy-match threshold for detection
  cooldown: "30m"                    # per-streamer notification cooldown
  workers: 2                         # subscriptions processed in parallel

steam:
  api_key: "..."                     # Steam Web API key (notifications)
  retries: 1                         # retries for Steam API calls

prune:
  interval: "1h"
  hours: 3                           # data retention window (hyperfocusdbd.com keeps 3h)
```

## Disclaimer

Hyperfocus is an independent fan project. It is not affiliated with,
endorsed by, or sponsored by Twitch Interactive, Valve Corporation,
Behaviour Interactive, or their partners. "Dead by Daylight" and related
trademarks, logos, and game content belong to Behaviour Interactive; Steam
and related marks belong to Valve; Twitch and related marks belong to Twitch
Interactive. The project respects their rights and complies with the
respective API terms.

## Related

- [hyperfocus2-ocr](https://github.com/rofleksey/hyperfocus2-ocr) — OCR microservice for survivor name extraction
