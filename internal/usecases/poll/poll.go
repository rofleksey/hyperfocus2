// Package poll is the usecase that periodically records a snapshot of every
// live Dead by Daylight stream. For each stream it captures a CDN preview
// image. Previews are fetched concurrently.
package poll

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/clock"
)

// StreamsGateway is the port for fetching live streams.
type StreamsGateway interface {
	GetLiveGameStreams(ctx context.Context) ([]entity.LiveStream, error)
}

// Repository is the persistence port needed by the poll usecase.
type Repository interface {
	UpsertStreamer(ctx context.Context, s entity.Streamer) error
	EnsureOpenSession(ctx context.Context, streamerID, twitchStreamID string, startedAt time.Time) (int64, *string, error)
	CloseUnseenSessions(ctx context.Context, seenIDs []string, now time.Time) (int64, error)
	InsertSnapshot(ctx context.Context, takenAt time.Time, source string, count int) (int64, error)
	InsertSample(ctx context.Context, s entity.StreamSample) error
	RunInTx(ctx context.Context, f func(ctx context.Context) error) error
}

// PreviewStore captures and persists a preview image.
type PreviewStore interface {
	FetchAndSave(ctx context.Context, url string) (string, error)
	Path(filename string) string
}

// OCRGateway extracts survivor usernames from preview images in a single batch.
// It is invoked once per poll cycle after previews have been captured.
type OCRGateway interface {
	ExtractSurvivors(ctx context.Context, paths []string) (map[string][]string, error)
}

// Deps holds the collaborators for New.
type Deps struct {
	Clock     clock.Clock
	Logger    *slog.Logger
	Gateway   StreamsGateway
	Repo      Repository
	Preview   PreviewStore
	OCR       OCRGateway
	Config    config.Poll
	OCRConfig config.OCR
}

// Poll is the polling usecase.
type Poll struct {
	clock   clock.Clock
	log     *slog.Logger
	gateway StreamsGateway
	repo    Repository
	prev    PreviewStore
	ocr     OCRGateway
	cfg     config.Poll
	ocrCfg  config.OCR
}

// New builds a Poll from its dependencies.
func New(d Deps) *Poll {
	if d.Clock == nil {
		d.Clock = clock.System()
	}
	return &Poll{
		clock: d.Clock, log: d.Logger, gateway: d.Gateway,
		repo: d.Repo, prev: d.Preview, ocr: d.OCR, cfg: d.Config, ocrCfg: d.OCRConfig,
	}
}

// Run polls forever until ctx is cancelled, running cycles back-to-back with
// no delay between them.
func (p *Poll) Run(ctx context.Context) {
	p.log.Info("poll loop starting")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		p.tick(ctx)
	}
}

func (p *Poll) tick(ctx context.Context) {
	if err := p.doPoll(ctx); err != nil {
		p.log.Error("poll cycle failed", slog.Any("error", err))
	}
}

type streamResult struct {
	stream        entity.LiveStream
	sessionID     int64
	previewFile   string
	thumbFile     string
	survivorNames []string
}

func (p *Poll) doPoll(ctx context.Context) error {
	started := time.Now()
	now := p.clock.Now().UTC()

	p.log.Debug("poll: cycle start")

	fetchStart := time.Now()
	rawStreams, err := p.fetchWithRetry(ctx)
	fetchDur := time.Since(fetchStart)
	if err != nil {
		return oops.Wrap(err)
	}
	streams := dedupStreams(rawStreams)
	if dups := len(rawStreams) - len(streams); dups > 0 {
		p.log.Debug("poll: dropped duplicate streams from fetch", slog.Int("duplicates", dups))
	}
	if len(streams) == 0 {
		p.log.Info("poll: no live streams")
		return nil
	}

	p.log.Debug("poll: streams fetched",
		slog.Int("raw", len(rawStreams)),
		slog.Int("deduped", len(streams)))

	// -----------------------------------------------------------------------
	// Phase 2 — ensure sessions.
	// -----------------------------------------------------------------------
	results := make([]streamResult, len(streams))
	for i, s := range streams {
		results[i].stream = s
	}
	var snapshotID int64

	if err := p.repo.RunInTx(ctx, func(tctx context.Context) error {
		for i, st := range streams {
			if err := p.repo.UpsertStreamer(tctx, entity.Streamer{
				TwitchUserID: st.TwitchUserID,
				Login:        st.Login,
				DisplayName:  st.DisplayName,
				LastSeen:     now,
			}); err != nil {
				return oops.Wrap(err)
			}
			sid, _, err := p.repo.EnsureOpenSession(tctx, st.TwitchUserID, st.TwitchStreamID, st.StartedAt)
			if err != nil {
				return oops.Wrap(err)
			}
			results[i].sessionID = sid
		}
		return nil
	}); err != nil {
		return err
	}

	p.log.Debug("poll: sessions created", slog.Int("total", len(results)))

	// -----------------------------------------------------------------------
	// Phase 3 — capture previews. OCR survivor names are extracted in
	// parallel with downloads: every successfully saved preview is fed into a
	// queue consumed by N OCR worker goroutines, so OCR on an image starts as
	// soon as a worker is free. Both downloads and OCR are waited on before
	// the cycle proceeds.
	// -----------------------------------------------------------------------
	captureStart := time.Now()
	var downloadDur, ocrDur time.Duration

	if p.ocr != nil && p.ocrCfg.IsEnabled() {
		downloadDur, ocrDur = p.captureAndOCR(ctx, results)
	} else {
		downloadDur = p.captureAll(ctx, results, nil)
	}
	captureDur := time.Since(captureStart)

	previewOk := 0
	survivorsFound := 0
	for _, r := range results {
		if r.previewFile != "" {
			previewOk++
		}
		if len(r.survivorNames) > 0 {
			survivorsFound++
		}
	}
	p.log.Debug("poll: capture complete",
		slog.Int("streams", len(results)),
		slog.Int("previews_ok", previewOk),
		slog.Duration("duration", captureDur))

	// -----------------------------------------------------------------------
	// Phase 4 — second DB tx: snapshot + close unseen.
	// -----------------------------------------------------------------------
	seen := make([]string, 0, len(streams))
	if err := p.repo.RunInTx(ctx, func(tctx context.Context) error {
		for _, r := range results {
			seen = append(seen, r.stream.TwitchStreamID)
		}
		id, err := p.repo.InsertSnapshot(tctx, now, "twitch", len(streams))
		if err != nil {
			return oops.Wrap(err)
		}
		snapshotID = id
		if _, err := p.repo.CloseUnseenSessions(tctx, seen, now); err != nil {
			return oops.Wrap(err)
		}
		return nil
	}); err != nil {
		return err
	}

	p.log.Debug("poll: sessions tx committed",
		slog.Int64("snapshot_id", snapshotID),
		slog.Int("sessions", len(results)))

	// -----------------------------------------------------------------------
	// Phase 5 — third DB tx: samples with preview.
	// -----------------------------------------------------------------------
	if err := p.repo.RunInTx(ctx, func(tctx context.Context) error {
		for _, r := range results {
			var fn *string
			if r.previewFile != "" {
				f := r.previewFile
				fn = &f
			}
			var tn *string
			if r.thumbFile != "" {
				t := r.thumbFile
				tn = &t
			}
			if err := p.repo.InsertSample(tctx, entity.StreamSample{
				SnapshotID:    snapshotID,
				SessionID:     r.sessionID,
				StreamerID:    r.stream.TwitchUserID,
				ViewerCount:   r.stream.ViewerCount,
				Title:         r.stream.Title,
				Language:      r.stream.Language,
				Tags:          r.stream.Tags,
				StartedAt:     r.stream.StartedAt,
				PreviewFilename: fn,
				ThumbFilename:   tn,
				SurvivorNames: r.survivorNames,
			}); err != nil {
				return oops.Wrap(err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	totalDur := time.Since(started)

	p.log.Info("poll cycle complete",
		slog.Int("streams", len(streams)),
		slog.Int64("snapshot_id", snapshotID),
		slog.Int("previews", previewOk),
		slog.Int("previews_failed", len(results)-previewOk),
		slog.Int("survivors_found", survivorsFound),
		slog.Duration("fetch_duration", fetchDur),
		slog.Duration("download_duration", downloadDur),
		slog.Duration("ocr_duration", ocrDur),
		slog.Duration("capture_duration", captureDur),
		slog.Duration("total_duration", totalDur),
	)
	return nil
}

// ---------------------------------------------------------------------------
// fetchWithRetry
// ---------------------------------------------------------------------------

func (p *Poll) fetchWithRetry(ctx context.Context) ([]entity.LiveStream, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts(p.cfg.FetchMaxAttempts); attempt++ {
		p.log.Debug("poll: twitch api call (streams)", slog.Int("attempt", attempt))
		streams, err := p.gateway.GetLiveGameStreams(ctx)
		if err == nil {
			return streams, nil
		}
		lastErr = err
		p.log.Warn("poll: fetch streams failed; retrying",
			slog.Int("attempt", attempt), slog.Any("error", err))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.cfg.FetchDelay.Std()):
		}
	}
	return nil, oops.Wrap(lastErr)
}

// ---------------------------------------------------------------------------
// captureAndOCR — captureAll + parallel OCR via a worker pool over a channel.
// Each downloaded preview path is sent into a channel. N workers pull from it,
// accumulating small batches (ocrWorkerBatchSize images) and dispatching each
// batch to a single OCR subprocess. Batching amortizes the ~200ms ONNX model
// load per python process start; the small batch size keeps latency low so OCR
// overlaps well with downloads. The method blocks until all downloads and OCR
// work are finished.
// ---------------------------------------------------------------------------

const ocrWorkerBatchSize = 50

func (p *Poll) captureAndOCR(ctx context.Context, results []streamResult) (time.Duration, time.Duration) {
	previewCount := len(results)
	if previewCount == 0 {
		return 0, 0
	}

	started := time.Now()
	ocrPaths := make(chan string, previewCount)

	merged := make(map[string][]string, previewCount)
	var mu sync.Mutex
	var ocrWg sync.WaitGroup

	workers := p.ocrCfg.Workers
	if workers <= 0 {
		workers = 1
	}

	p.log.Info("ocr: worker pool started",
		slog.Int("expected_images", previewCount),
		slog.Int("workers", workers),
		slog.Int("python_workers", p.ocrCfg.PythonWorkers),
		slog.Int("batch_size", ocrWorkerBatchSize))

	for w := 0; w < workers; w++ {
		ocrWg.Add(1)
		go func() {
			defer ocrWg.Done()
			batch := make([]string, 0, ocrWorkerBatchSize)
			flush := func() {
				if len(batch) == 0 {
					return
				}
				b := make([]string, len(batch))
				copy(b, batch)
				batch = batch[:0]
				namesByPath, err := p.ocr.ExtractSurvivors(ctx, b)
				if err != nil {
					p.log.Warn("ocr: batch failed",
						slog.Any("error", err),
						slog.Int("images", len(b)))
					return
				}
				mu.Lock()
				for k, v := range namesByPath {
					merged[k] = v
				}
				mu.Unlock()
			}
			for path := range ocrPaths {
				batch = append(batch, path)
				if len(batch) >= ocrWorkerBatchSize {
					flush()
				}
			}
			flush()
		}()
	}

	p.captureAll(ctx, results, ocrPaths)
	downloadDur := time.Since(started)
	p.log.Info("poll: downloads finished",
		slog.Int("images", previewCount),
		slog.Duration("duration", downloadDur))
	close(ocrPaths)
	ocrWg.Wait()
	ocrDur := time.Since(started)

	p.log.Info("ocr: worker pool finished",
		slog.Int("images_total", previewCount),
		slog.Int("samples_with_names", len(merged)),
		slog.Duration("download_duration", downloadDur),
		slog.Duration("ocr_duration", ocrDur))

	for i := range results {
		if results[i].previewFile == "" {
			continue
		}
		names := merged[p.prev.Path(results[i].previewFile)]
		if len(names) > 0 {
			results[i].survivorNames = names
		}
	}
	return downloadDur, ocrDur
}

// ---------------------------------------------------------------------------
// captureAll — concurrent preview download for every stream.
// When ocrPaths is non-nil, each successfully saved preview path is sent to it
// so OCR can run in parallel with the remaining downloads.
// ---------------------------------------------------------------------------

func (p *Poll) captureAll(ctx context.Context, results []streamResult, ocrPaths chan<- string) time.Duration {
	if len(results) == 0 {
		return 0
	}

	started := time.Now()
	workers := p.cfg.PreviewWorkers
	if workers <= 0 {
		workers = 8
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := &results[i]

			sem <- struct{}{}

			// Preview — full resolution (for OCR + modal).
			previewURL := buildPreviewURL(r.stream.ThumbnailURL, p.cfg.PreviewWidth, p.cfg.PreviewHeight)
			if previewURL != "" {
				fctx, cancel := context.WithTimeout(ctx, p.cfg.PreviewTimeout.Std())
				fn, err := p.prev.FetchAndSave(fctx, previewURL)
				cancel()
				if err != nil {
					p.log.Warn("poll: preview failed",
						slog.String("login", r.stream.Login),
						slog.String("url", previewURL),
						slog.Any("error", err))
				} else {
					r.previewFile = fn
					if ocrPaths != nil {
						ocrPaths <- p.prev.Path(fn)
					}
				}
			}

			// Thumb — low resolution (for gallery grid).
			if r.previewFile != "" {
				thumbURL := buildPreviewURL(r.stream.ThumbnailURL, p.cfg.ThumbPreviewWidth, p.cfg.ThumbPreviewHeight)
				if thumbURL != "" {
					tctx, cancel := context.WithTimeout(ctx, p.cfg.PreviewTimeout.Std())
					tn, err := p.prev.FetchAndSave(tctx, thumbURL)
					cancel()
					if err != nil {
						p.log.Warn("poll: thumb preview failed",
							slog.String("login", r.stream.Login),
							slog.Any("error", err))
					} else {
						r.thumbFile = tn
					}
				}
			}

			<-sem
		}(i)
	}
	wg.Wait()
	return time.Since(started)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildPreviewURL(template string, w, h int) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	if w <= 0 {
		w = 640
	}
	if h <= 0 {
		h = 360
	}
	out := strings.ReplaceAll(template, "{width}", itoa(w))
	out = strings.ReplaceAll(out, "{height}", itoa(h))
	return out
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

func maxAttempts(n int) int {
	if n <= 0 {
		return 3
	}
	return n
}

func dedupStreams(in []entity.LiveStream) []entity.LiveStream {
	seen := make(map[string]struct{}, len(in))
	out := make([]entity.LiveStream, 0, len(in))
	for _, s := range in {
		if s.TwitchStreamID == "" {
			continue
		}
		if _, ok := seen[s.TwitchStreamID]; ok {
			continue
		}
		seen[s.TwitchStreamID] = struct{}{}
		out = append(out, s)
	}
	return out
}
