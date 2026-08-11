<p align="center">
  <img src="screenshot.png" alt="Hyperfocus screenshot" width="720">
</p>

<h3 align="center"><a href="https://hyperfocusdbd.com">🔗 hyperfocusdbd.com</a></h3>

# Hyperfocus — Dead by Daylight live stream history

Hyperfocus tracks every active [Dead by Daylight](https://store.steampowered.com/app/381210/Dead_by_Daylight/) stream on Twitch in real time. It captures periodic snapshots of viewer counts, stream titles, preview thumbnails, and player names so you can browse who was online at any moment — past or present.

## Features

- **Live polling** — continuously fetches all DBD streams from Twitch (~2000 per cycle) in a back-to-back poll loop
- **Historical snapshots** — navigate back in time with prev/next buttons, a datetime picker, or a "Now" button with new-data indicator
- **Thumbnail gallery** — responsive grid of 16:9 stream previews with viewer counts, relevance scores, and infinite scroll
- **OCR survivor names** — extracts the 4 player names from the bottom-left HUD of each stream's preview using [PaddleOCR on ONNX Runtime](https://github.com/rofleksey/hyperfocus2-ocr)
- **Fuzzy search** — search by streamer name, language, or survivor in-game name with a loose matcher that tolerates OCR recognition errors
- **Streamer detail** — click any streamer to see their sample at that moment: preview, viewer count, title, tags, and OCR survivor names
- **Stats charts** — line charts for streams online, total viewers, cycle time, disk usage, preview capture rate, and OCR success rate over time
- **Notifications** — optional Twitch chat bot that notifies streamers when a tracked player appears in their lobby
- **No accounts, no cookies** — built with Vue 3 + PrimeVue, dark theme, responsive

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

ocr:
  enabled: true
  api_url: "http://localhost:8081"   # hyperfocus2-ocr service

notify:
  enabled: false                     # optional chat notifications

prune:
  interval: "1h"
  hours: 72                          # data retention window (default)
```

## Related

- [hyperfocus2-ocr](https://github.com/rofleksey/hyperfocus2-ocr) — OCR microservice for survivor name extraction
