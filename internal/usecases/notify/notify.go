package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/fuzzy"
)

type Repository interface {
	ActiveSubscribers(ctx context.Context) ([]entity.NotificationSubscriber, error)
	RecentNotification(ctx context.Context, subscriberID int64, detectedName string, cooldown time.Duration) (bool, error)
	LogNotification(ctx context.Context, subscriberID int64, detectedName string, score float64, snapshotID int64, sourceStreamerID string) error
	UpdateSteamNames(ctx context.Context, names map[int64]string) error
}

type IRCMessenger interface {
	Send(ctx context.Context, channel, message string)
}

type SteamNameResolver interface {
	GetPlayerSummaries(ctx context.Context, steamIDs []string) ([]string, error)
}

type Deps struct {
	Logger    *slog.Logger
	Repo      Repository
	IRC       IRCMessenger
	Steam     SteamNameResolver
	BotUserID string
	Config    config.Notify
}

type Service struct {
	log       *slog.Logger
	repo      Repository
	irc       IRCMessenger
	steam     SteamNameResolver
	botUserID string
	cfg       config.Notify

	mu         sync.RWMutex
	cache      []cachedSubscriber
	cacheUntil time.Time
	running    atomic.Bool
}

type cachedSubscriber struct {
	ID           int64
	TwitchLogin  string
	TwitchUserID string
	SteamID      string
	SteamName    string
}

func New(d Deps) *Service {
	return &Service{
		log:       d.Logger,
		repo:      d.Repo,
		irc:       d.IRC,
		steam:     d.Steam,
		botUserID: d.BotUserID,
		cfg:       d.Config,
	}
}

// ProcessAsync schedules a snapshot for notification processing on its own
// goroutine so the poll cycle never blocks on the Steam API. If the previous
// run is still in flight, the snapshot is skipped (notifications are only
// meaningful for fresh data anyway).
func (s *Service) ProcessAsync(ctx context.Context, snapID int64, samples []entity.StreamSample) {
	if !s.cfg.IsEnabled() {
		return
	}
	if !s.running.CompareAndSwap(false, true) {
		s.log.Debug("notify: previous cycle still processing; skipping")
		return
	}
	go func() {
		defer s.running.Store(false)
		s.ProcessSnapshot(ctx, snapID, samples)
	}()
}

func (s *Service) ProcessSnapshot(ctx context.Context, snapID int64, samples []entity.StreamSample) {
	if !s.cfg.IsEnabled() {
		return
	}

	subs := s.activeSubscribers(ctx)
	if len(subs) == 0 {
		return
	}

	matchNames := s.fetchMatchNames(ctx, subs)
	if len(matchNames) == 0 {
		return
	}

	workers := s.cfg.Workers
	if workers <= 0 {
		workers = 2
	}

	tasks := make(chan cachedSubscriber)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sub := range tasks {
				s.processSubscription(ctx, sub, samples, snapID, matchNames)
			}
		}()
	}
	for _, sub := range subs {
		tasks <- sub
	}
	close(tasks)
	wg.Wait()
}

// processSubscription matches one subscriber against every sample and sends
// notifications when their Steam name shows up in another streamer's lobby.
func (s *Service) processSubscription(ctx context.Context, sub cachedSubscriber, samples []entity.StreamSample, snapID int64, matchNames map[int64][]string) {
	names, ok := matchNames[sub.ID]
	if !ok {
		return
	}
	for _, sample := range samples {
		if sample.StreamerLogin == "" || sub.TwitchUserID == sample.StreamerID {
			continue
		}
		best := 0.0
		for _, name := range names {
			for _, survivor := range sample.SurvivorNames {
				if score := fuzzy.Score(name, survivor); score > best {
					best = score
				}
			}
		}
		if best < s.cfg.MinScore {
			continue
		}
		recent, err := s.repo.RecentNotification(ctx, sub.ID, sample.StreamerLogin, s.cfg.Cooldown.Std())
		if err != nil {
			s.log.Warn("notify: dedup check failed", slog.Any("error", err))
			continue
		}
		if recent {
			continue
		}
		msg := fmt.Sprintf("You might be playing with @%s", sample.StreamerLogin)
		s.irc.Send(ctx, sub.TwitchLogin, msg)
		if err := s.repo.LogNotification(ctx, sub.ID, sample.StreamerLogin, best, snapID, sample.StreamerID); err != nil {
			s.log.Warn("notify: log failed", slog.Any("error", err))
		}
		s.log.Info("notify: sent",
			slog.String("to", sub.TwitchLogin),
			slog.String("streamer", sample.StreamerLogin),
			slog.Float64("score", best))
	}
}

// fetchMatchNames resolves the current Steam persona name for every
// subscriber. On success the fresh names are persisted so they can serve as a
// fallback later; only when the API fails entirely (all retries exhausted)
// are the stored names used.
func (s *Service) fetchMatchNames(ctx context.Context, subs []cachedSubscriber) map[int64][]string {
	steamIDs := make([]string, 0, len(subs))
	for _, sub := range subs {
		if sub.SteamID != "" {
			steamIDs = append(steamIDs, sub.SteamID)
		}
	}
	if len(steamIDs) == 0 {
		return nil
	}

	names, err := s.steam.GetPlayerSummaries(ctx, steamIDs)
	if err != nil {
		s.log.Warn("notify: steam api failed, using stored names", slog.Any("error", err))
		out := make(map[int64][]string)
		for _, sub := range subs {
			if nn := fuzzy.Norm(sub.SteamName); nn != "" {
				out[sub.ID] = []string{nn}
			}
		}
		return out
	}

	out := make(map[int64][]string)
	updates := make(map[int64]string)
	for i, sub := range subs {
		if i >= len(names) {
			continue
		}
		if names[i] == "" {
			continue
		}
		updates[sub.ID] = names[i]
		out[sub.ID] = []string{fuzzy.Norm(names[i])}
	}
	if err := s.repo.UpdateSteamNames(ctx, updates); err != nil {
		s.log.Warn("notify: failed to store steam names", slog.Any("error", err))
	}
	return out
}

func (s *Service) activeSubscribers(ctx context.Context) []cachedSubscriber {
	s.mu.RLock()
	if time.Now().Before(s.cacheUntil) {
		c := s.cache
		s.mu.RUnlock()
		return c
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().Before(s.cacheUntil) {
		return s.cache
	}

	subs, err := s.repo.ActiveSubscribers(ctx)
	if err != nil {
		s.log.Warn("notify: failed to load subscribers", slog.Any("error", err))
		return s.cache
	}

	out := make([]cachedSubscriber, 0, len(subs))
	for _, sub := range subs {
		if sub.SteamID == "" {
			continue
		}
		out = append(out, cachedSubscriber{
			ID:           sub.ID,
			TwitchLogin:  sub.TwitchLogin,
			TwitchUserID: sub.TwitchUserID,
			SteamID:      sub.SteamID,
			SteamName:    sub.SteamName,
		})
	}

	s.cache = out
	s.cacheUntil = time.Now().Add(2 * time.Minute)
	return out
}
