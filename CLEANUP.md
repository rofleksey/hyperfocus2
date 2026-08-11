# Hyperfocus2 — Cleanup & Improvement Plan

## Phase 1 — Emergency Security

### Secrets & Access
- [ ] `chmod 600 config.yaml` (currently world-readable `0644` with live secrets)

### Container Security
- [ ] Re-add non-root `USER 65532:65532` in Dockerfile (revert regression in `caf7262`)
- [ ] `chown` binary and `./data` directory to `65532:65532` in Dockerfile
- [ ] Add `--read-only --tmpfs /tmp` to Makefile `docker` target docs
- [ ] Pin base images by digest: `node:22-alpine`, `golang:1.26-alpine`, `alpine:3.20`

### HTTP Security
- [ ] Bound JSON request bodies on POST / DELETE `/api/subscribe` via `http.MaxBytesReader`
- [ ] Fix `httputil.DecodeJSON` — pass `http.ResponseWriter` to `MaxBytesReader` (currently `nil`)
- [ ] Add `Origin` / `Sec-Fetch-Site` header check on `/api/subscribe` write operations (CSRF)
- [ ] Remove CORS middleware entirely — SPA is same-origin; no `Access-Control-Allow-*` headers needed
- [ ] Add `MaxHeaderBytes` (`1 << 16`) to `http.Server` config
- [ ] Add `IdleTimeout` to `http.Server` config

### Frontend Security
- [ ] Add `rel="noopener noreferrer"` to all `target="_blank"` anchors (MomentView gallery, StreamDetailView, AboutView)
- [ ] Add `<meta http-equiv="Content-Security-Policy">` tag to `web/index.html`

---

## Phase 2 — Critical Correctness Bugs

### Shutdown / Lifecycle
- [ ] Fix shutdown leak: wrap `app.Shutdown()` in `defer` after `select`; remove early `return err` in `main.go`
- [ ] Make `IRCBot.Run` cancellable: call `c.Disconnect()` in a ctx-done watcher inside `runOnce`
- [ ] Replace `time.Sleep(10s)` in `IRCBot.runOnce` with `select { case <-time.After(...); case <-ctx.Done(): return }`
- [ ] Tie IRC join-goroutine lifetime to per-iteration done channel (avoid stale-client joins on reconnect)
- [ ] `RunInTx.Rollback`: use `context.Background()` instead of possibly-cancelled parent ctx
- [ ] Log error when `ActiveSubscriberChannels` fails instead of silent `_`

### Data Races
- [ ] Serialize all `BotHelix.helix` calls behind a mutex (`SetUserAccessToken` is called concurrently)
- [ ] Serialize `BotTokenStore.Refresh` with per-refresh mutex (two concurrent refreshes rotate each other's token)

### Error Handling
- [ ] Translate `pgx.ErrNoRows` → `entity.ErrNotFound` in all repository finder methods
- [ ] Handler maps `errors.Is(err, entity.ErrNotFound)` → HTTP 404 (currently 500)
- [ ] Detect `pgconn.ErrUniqueViolation` → HTTP 409 on `/api/subscribe` TOCTOU race
- [ ] Use `httputil.DecodeJSON` (fixed) in subscribe handlers instead of bare `json.NewDecoder`

### HTTP Client Timeouts
- [ ] Replace `http.DefaultClient` with configured client in `pkg/twitch/token.go` token refresh
- [ ] Make `pkg/steam/steam.go` `GetPlayerSummaries` use `c.client` (has `15s` timeout) instead of `http.DefaultClient`

### API Correctness
- [ ] Add inter-cycle delay to `Poll.Run` (`time.NewTicker(30s)` or similar)
- [ ] Double-checked locking: re-check `cacheUntil` after acquiring write lock in `activeSubscribers`
- [ ] Make `notify.ProcessSnapshot` async (`go s.ProcessSnapshot(...)`) to not block poll cycle
- [ ] Document captured-index invariant in `captureAndOCR` (each worker writes distinct `results[i]`)

---

## Phase 3 — Migrations Squash & Schema Cleanup

### Squash
- [ ] Collapse migrations `0001` through `0005` into a single `0001_init.sql`
- [ ] Delete files `0002_stats.sql`, `0003_subscriber_names.sql`, `0004_notify.sql`, `0005_notify_cascade.sql`

### Drop Unused Schema
- [ ] Drop `vods` table (VOD fetching was removed; table is dead weight)
- [ ] Drop `stream_sessions.vod_id` and `stream_sessions.vod_created_at` columns
- [ ] Drop `stream_samples.vod_offset_seconds` column
- [ ] Drop `tags TEXT[]` column from `stream_samples` (never queried; no index)
- [ ] Drop `subscriber_names` table (never read; every insert passes `nil`)
- [ ] Remove duplicate `CREATE TABLE schema_version` from `0001_init.sql` (already in `migrations.go`)

### Add Missing Indexes
- [ ] Add `CREATE INDEX idx_notif_log_snapshot ON notification_log(snapshot_id)` (needed for `ON DELETE CASCADE` prune)
- [ ] Add `CREATE INDEX idx_notif_log_source ON notification_log(source_streamer_id)`
- [ ] Add `CREATE INDEX idx_samples_survivors` GIN on `survivor_names` (keep existing)
- [ ] Add `CREATE INDEX idx_streamer_login_trgm ON streamers USING GIN (login gin_trgm_ops)` for fuzzy search
- [ ] Add `CREATE INDEX idx_notif_subscribers_active ON notification_subscribers(active) WHERE active = TRUE`

### Runner Improvements
- [ ] Inject `*slog.Logger` into migration runner instead of `fmt.Printf`
- [ ] Add checksum column to `schema_version` and verify on startup

---

## Phase 4 — CI / Infrastructure

### CI Workflows
- [ ] Add `test` job: `go test -race -cover ./...` with `setup-go@v5` + module cache
- [ ] Add `golangci-lint-action` step (`.golangci.yaml` already configured)
- [ ] Add frontend `npm run typecheck` step to CI
- [ ] Add `permissions: { contents: read, packages: write }` to `build-push.yml`
- [ ] Add `concurrency: { group: build-${{ github.ref }}, cancel-in-progress: true }` to `build-push.yml`
- [ ] Gate `push: true` to default branch only (feature branches build but don't push)
- [ ] Add `docker/metadata-action` version tag derived from git tag/sha
- [ ] Add BuildKit cache mounts (`--mount=type=cache,target=/root/.cache/go-build`, `/go/pkg/mod`) to Dockerfile
- [ ] Add `cache-from: type=gha` / `cache-to: type=gha,mode=max` to build-push action
- [ ] Add Trivy vulnerability scan step (fail on CRITICAL)
- [ ] Add cosign image signing step
- [ ] Add SBOM generation step

### Dockerfile
- [ ] Inject `ARG VERSION=dev` + ldflags `-X 'hyperfocus/internal/container.Version=${VERSION}'`
- [ ] Replace `apk add curl` with built-in `wget` for HEALTHCHECK (or `/api/healthz` TCP check)
- [ ] Add minimal OCI labels in Dockerfile (`org.opencontainers.image.source`, `description`)

### Makefile
- [ ] Add `help` target (self-documenting) as default
- [ ] Add `test` target: `go test -race -cover ./...`
- [ ] Fix `lint` target: fail loudly when `golangci-lint` is missing (remove `|| echo "skipping"`)
- [ ] Add `typecheck` target: `cd web && npm run typecheck`
- [ ] Cache `web/node_modules` with Makefile dependency rule (only re-run `npm ci` when `package-lock.json` changes)
- [ ] Split `clean` into `clean` (artifacts only) and `distclean` (also `node_modules`)

### Configuration
- [ ] Add env-var overrides: `HYPERFOCUS_CONFIG` path, `HYPERFOCUS_TWITCH_CLIENT_SECRET`, etc.
- [ ] Default `ssl_mode` to `"prefer"` instead of `"disable"`
- [ ] Validate `ssl_mode` against allowlist (`disable|allow|prefer|require|verify-ca|verify-full`)
- [ ] Add bounded validators: `PageSize` (`min=1,max=1000`), `PreviewWorkers` (`min=1,max=128`), `FetchMaxAttempts` (`min=1,max=20`)
- [ ] Enable `yaml.Decoder.KnownFields(true)` to reject unknown config keys (prevents silent ignore)
- [ ] Fix `config.yaml`: rename `prune.days: 3` → `prune.hours: 72`
- [ ] Document `notify`, `steam`, `twitch_bot` sections in `config.example.yaml`
- [ ] Remove unused `Poll.PageDelay` and `Poll.PageSize` config keys (never read) or wire them through

### .dockerignore
- [ ] Add `README.md`, `screenshot.png`, `*.md`, `Makefile`, `.golangci.yaml`, `.editorconfig`, `scripts/`
- [ ] Add `*.pem`, `*.key`, `*.p12`, `.env*`, `secrets/`, `CLEANUP.md`

### Misc
- [ ] Remove empty `scripts/` directory

---

## Phase 5 — Performance

### Backend
- [ ] Replace `dirSize` filepath.Walk (every poll cycle) with atomic counter (inc on save, dec on sweep)
- [ ] Fix O(N·M) steamRefreshLoop: build `map[steamID]*sub` before batch loop; batch UPDATEs
- [ ] Escape ILIKE wildcards (`%`, `_`, `\`) in user-supplied `q` with `ESCAPE '\'`
- [ ] Cap unbounded `findLimit` to `1000` when survivor/query/language filters are set
- [ ] Tune pgx pool: add `MaxConnIdleTime=30m`, `MinConns=2`, `HealthCheckPeriod=1m`
- [ ] Raise OCR worker default to `max(2, preview_workers/4)` instead of hardcoded `1`
- [ ] `io.LimitReader` on OCR response JSON decode (unbounded body from microservice)
- [ ] Run `SnapshotAtOrBefore` / `SnapshotAtOrAfter` concurrently with `errgroup`

### Frontend Bundle
- [ ] Lazy-load all routes: `component: () => import("./views/...")` (splits ~784 KB into chunks)
- [ ] Tree-shake Chart.js: import only `LineController`, `LineElement`, `PointElement`, scales, tooltip, legend, filler

### Frontend Rendering
- [ ] Reduce `fetchSnapshots(1000)` → `200`; lazy-load older snapshots on prev navigation
- [ ] Add `v-memo` on gallery items keyed on `[thumb_url, fuzzy_score, at]`
- [ ] Memoize `scoreColor` / `scorePct` per stream (WeakMap or precompute on fetch)
- [ ] Add `content-visibility: auto` and `contain-intrinsic-size` to gallery items
- [ ] Add `decoding="async"` and `loading="lazy"` to all thumbnail `<img>` tags
- [ ] Replace `primeicons` font (~640 KB) with inline SVG icons (lucide-vue-next or subsetted font)
- [ ] Convert `skill-issue.gif` (816 KB) to WebP/MP4 or lazy-load

### Polling / Requests
- [ ] Pause `checkLatest` interval when `document.visibilityState === "hidden"`
- [ ] Add request deduplication: skip `loadFirstPage` if same params are already in-flight
- [ ] Combine two `watch(at, ...)` watchers into one: `watch(at, () => { syncURL(); debounceLoad(); })`
- [ ] Remove redundant `watch(allStreams, ...)` observer watcher (sentinel watch is sufficient)

---

## Phase 6 — Frontend Bugs

### Data Fetching
- [ ] Add `AbortController` support to `api.ts`; abort previous in-flight request on each new call
- [ ] Clear `allStreams` at start of `loadFirstPage` (avoid stale data + loading spinner simultaneously)
- [ ] When `loadFirstPage` fails, surface error (currently swallowed silently by outer `try/catch`)

### URL / State Sync
- [ ] Fix `syncingFromRoute` flag: use `flush: "sync"` watcher + `await nextTick()` pattern
- [ ] Validate `sort` and `dir` query params against allowlists (`["viewers","name","started"]`, `["asc","desc"]`)
- [ ] Use `qStr()` helper for all `route.query` access (handle `string | string[] | null | undefined` type)
- [ ] Fix `StreamDetailView` watcher: remove dead `?? route.params.streamer_id` branch; add `route.query.at` to watch deps

### Pagination
- [ ] Handle pagination errors: show loadMore error, reset `hasMore` on failure, add retry button
- [ ] Fix `IntersectionObserver` callback: use `entries.some(e => e.isIntersecting)`
- [ ] Throttle IntersectionObserver callback (guard against synchronous multi-fire)

### Navigation
- [ ] Replace gallery `<a target="_blank">` with `<RouterLink>` for in-app navigation (default click)
- [ ] NotFoundView: use `<RouterLink to="/">` instead of `<a href="/">` (avoids full page reload)
- [ ] Scroll to top on `loadFirstPage` success (`window.scrollTo({ top: 0 })`)

### Subscribe Form
- [ ] Check `Content-Type` before JSON.parse; check `r.ok` before parsing (non-JSON error pages crash)
- [ ] Re-validate `twitchLogin.value.trim()` and `steamURL.value.trim()` before issuing request
- [ ] Add client-side regex validation: Twitch username `^[a-zA-Z0-9_]{4,25}$`, Steam URL format
- [ ] Show loading indicator on `checkStatus` (@blur fetcher)
- [ ] Wrap status banner in `<div role="status" aria-live="polite">`

### Memory & Cleanup
- [ ] Clear `debounceTimer` in `onUnmounted` (state-set-on-unmounted leak)
- [ ] Remove redundant `watch(allStreams, ...)` observer watcher

### Hardcoded Values
- [ ] Centralize retention constant (`RETENTION_HOURS`) — used in 3 places with different text
- [ ] Centralize magic numbers: `PAGE_SIZE`, `DEBOUNCE_MS`, `POLL_INTERVAL_MS`, `FETCH_SNAPSHOTS_LIMIT`
- [ ] Centralize default page limits to `config.ts`

### Embed
- [ ] Document `npm run build` prerequisite in `embed.go`; add build-time guard or sentinel file

---

## Phase 7 — A11y & UX

### Accessibility (A11y)
- [ ] Add `aria-label` to all icon-only buttons (App header nav, moment controls, clear × buttons)
- [ ] Convert clear `×` `<span>` → `<button type="button" aria-label="Clear">`
- [ ] Add `<label for="survivor-search">` or `aria-label="Search survivors"` to survivor input
- [ ] Add `aria-hidden="true"` to all decorative icons
- [ ] Add `<h1>` to every page (currently only NotFoundView has one; headings jump from implicit to `<h2>`)
- [ ] Add skip-to-content link (`<a href="#main" class="skip-link">`) as first element in `<body>`
- [ ] Add `id="main"` to `<main>` element
- [ ] Add `<nav aria-label="Primary">` to app header nav
- [ ] Use `<ul role="list">` / `<li>` for gallery grid (screen readers announce list item count)
- [ ] Add `:focus-visible { outline: 2px solid var(--p-primary-color); }` globally
- [ ] Add `aria-label="Newer snapshot available"` to now-button when glowing
- [ ] Add "(opens in a new tab)" visible/hidden text to external Twitch/VOD links

### Error Handling
- [ ] Add "Try again" / "Retry" button next to every error message (MomentView, StreamDetailView, StatsView, SubscribeView)
- [ ] Extract `<ErrorBanner :message="error" @retry="...">` component
- [ ] Log errors to `console.warn` in dev (`import.meta.env.DEV`) in every `catch` block

### Loading / Empty States
- [ ] Extract `<LoadingSpinner>` component (duplicated CSS 3×)
- [ ] Add "No streams found — try adjusting filters" empty-state CTA in MomentView
- [ ] Add skeleton/placeholder background for thumbnail images (reduce CLS)
- [ ] Set explicit `width`/`height` attributes on gallery `<img>` tags

### UX Polish
- [ ] Add Reset button in filters dialog (clear `q`, `language`, restore sort/dir defaults)
- [ ] Add time-range selector and refresh button to StatsView (not just hardcoded 100 snapshots)
- [ ] Increase StatsView chart height from `220px` → `360px`; reduce x-axis label density
- [ ] Add theme toggle (read `prefers-color-scheme`, persist to localStorage)
- [ ] Allow `.moment-controls` to `flex-wrap: wrap` on mobile (currently overflows)
- [ ] Add `<meta name="description">`, OG/Twitter card tags, `<link rel="icon">`, `<meta name="theme-color">`
- [ ] Fix inconsistent retention text: PrivacyView says "3 days", AboutView says "6 hours", code says `6`

### Mobile / Responsive
- [ ] Fix `.moment-controls` wrapping on narrow screens
- [ ] Replace Dialog inline `style="width:420px"` with responsive CSS class

---

## Phase 8 — Dead Code Removal & Refactoring

### Backend Dead Code
- [ ] Remove `Repository.GetSubscriberNames` (`postgres/notification.go:47-63`)
- [ ] Remove `Repository.SamplePreviewFilenames` (`postgres/sample.go:134-152`) — commented "unused"
- [ ] Remove `BotHelix.SendChatMessage` (`pkg/twitch/api.go:88-101`) — notifications use IRC, not Helix chat
- [ ] Remove `httputil.DecodeJSON` (or fix & use it — Phase 2 already opted to fix & use)
- [ ] Remove `absDur` dead function (`usecases/poll/poll.go:499-504`)
- [ ] Replace custom `itoa` with `strconv.Itoa` (`usecases/poll/poll.go:506-526`)
- [ ] Remove `entity.StreamSession.IsOpen()` — never called
- [ ] Remove `entity.LiveStream.GameID` — set but never read (all streams are pre-filtered)
- [ ] Remove `DBTX` interface (`postgres/repository.go:20-24`) — unsafe to remove; verify not used as constraint
- [ ] Remove `subscriber_names` entity + related code (or wire it — Phase 3 op)

### Frontend Dead Code
- [ ] Remove `fetchStreamers` and `fetchStreamer` functions + `StreamerSummary`/`StreamerSession` interfaces (`api.ts:64-84`)
- [ ] Remove dead CSS classes: `.toolbar`, `.toolbar .field`, `.toolbar label`, `.check-field`, `.preview-thumb`, `.streamer-cell`, `.streamer-cell img.avatar` (`style.css:63-108`)
- [ ] Remove empty `<script setup lang="ts"></script>` from `AboutView.vue`
- [ ] Remove unused `RouterLink`/`RouterView` imports from `App.vue` (or remove from `AboutView` — pick one style)

### Backend Refactoring
- [ ] Merge `Preview` and `Thumb` handlers → `serveImage(w, r, dir)` helper
- [ ] Extract `runPeriodic(ctx context.Context, interval time.Duration, fn func(context.Context))` for three loops in `container.go`
- [ ] Consolidate `writeJSON` (subscribe.go) + `httputil.JSON` (httputil.go) → single `httputil.JSON`
- [ ] Replace `setStr`/`setInt`/`setInt32` closures with generic `func set[T comparable](dst *T, def T)`
- [ ] Extract SQL helper `nameMatch(col, param)` for duplicated `ILIKE '%' || $X || '%'` pattern
- [ ] Unify `config.TwitchBot` and `pkg/twitch.BotConfig` (same fields, manual field copy between them)
- [ ] Refactor `panic` → `return error` in `server.New` and `frontendHandler`
- [ ] Remove CORS middleware (Phase 1 item; then middleware.go only has logging + recovery)
- [ ] Add `charset=utf-8` to `Content-Type: application/json` responses

### Frontend Component Extraction
- [ ] Extract `<StreamThumb :stream>` component (gallery item + StreamDetailView avatar/thumb share pattern)
- [ ] Extract `<ClearableInput v-model placeholder>` component (survivor + q + language inputs share pattern)
- [ ] Extract `<ErrorBanner :message @retry>` component (4 error messages are identical)
- [ ] Extract `<Field label htmlFor>` wrapper component (Filters dialog + Subscribe form share layout)
- [ ] Extract `<LoadingSpinner>` component (duplicated CSS 3×)
- [ ] Extract `<LegalLayout>` wrapper (PrivacyView + TermsView share ~30 lines of identical CSS)
- [ ] Extract `utils/date.ts`: `fmtDate()` (duplicated 3×)
- [ ] Extract `composables/usePolling.ts`: interval + visibility pause pattern
- [ ] Extract `composables/useURLSync.ts`: route.query ↔ ref sync pattern

### Configuration & Tooling
- [ ] Add ESLint + Prettier config (`eslint-plugin-vue`)
- [ ] Fix single-quote inconsistency in `SubscribeView.vue` (everything else uses double quotes)
- [ ] Add `"engines": { "node": "^20" }` to `web/package.json`
- [ ] Add `tsconfig.json` path aliases: `"@/*": ["src/*"]`
- [ ] Add Vitest config + unit tests for `api.ts` URL construction and URL sync logic

### Naming & Types
- [ ] Rename `initials()` → `firstInitial()` in StreamDetailView (returns 1 char, misleading plural name)
- [ ] Rename `streamer_id` prop → `streamerId` (camelCase Vue convention)
- [ ] Replace `Record<string, string>` return type of `scoreColor` with `import("vue").CSSProperties`
- [ ] Fix `initials` fallback chain: `"?"`.trim().charAt(0) always returns `"?"` — pointless
- [ ] Replace raw `<a href="/">` in NotFoundView with `<RouterLink to="/">`

### Other Refactoring
- [ ] Remove defensive `/previews/` path check in `frontendHandler` (router already matches `GET /previews/{filename}`; redundant)
- [ ] Grow `commandLoop` channel buffer from 32 → 128, or log drop warnings
- [ ] `trigramSet` map: replace `map[string]bool` → `map[string]struct{}` (micro-optimization)
- [ ] Fix `approx` dead assignment in `fuzzy_test.go:53` (remove or use it)
- [ ] `Poll.AfterCycle` metrics: log `ocrDur` even in the `else` branch (capture-only path)

---

## Quick Wins (Can Be Done in Any Phase)

- [ ] Remove duplicate `CREATE TABLE schema_version` from `0001_init.sql`
- [ ] Add `noopener noreferrer` to all `target=_blank` links (Frontend)
- [ ] Fix `config.yaml` `prune.days` → `prune.hours`
- [ ] Drop empty `scripts/` directory
- [ ] Tighten `.dockerignore`
- [ ] Add `charset=utf-8` to all `Content-Type: application/json` headers
- [ ] Remove unused `import` statements found by linter
- [ ] Log swallowed errors to console in frontend dev mode
- [ ] Add `aria-label` to Header nav buttons
- [ ] Fix `initials()` name + implementation in StreamDetailView
- [ ] Replace custom `itoa` with `strconv.Itoa`
- [ ] Remove `absDur` dead function
- [ ] Fix `approx` dead assignment in test file
