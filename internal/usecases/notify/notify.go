package notify

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/fuzzy"
)

type Repository interface {
	ActiveSubscribers(ctx context.Context) ([]entity.NotificationSubscriber, error)
	RecentNotification(ctx context.Context, subscriberID int64, detectedName string, cooldown time.Duration) (bool, error)
	LogNotification(ctx context.Context, subscriberID int64, detectedName string, score float64, snapshotID int64, sourceStreamerID string) error
}

type IRCMessenger interface {
	Send(channel, message string)
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
}

type cachedSubscriber struct {
	ID           int64
	TwitchLogin  string
	TwitchUserID string
	SteamID      string
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

	for _, sample := range samples {
		for _, survivor := range sample.SurvivorNames {
			for _, sub := range subs {
				if sub.TwitchUserID == sample.StreamerID {
					continue
				}
				names, ok := matchNames[sub.ID]
				if !ok {
					continue
				}
				for _, name := range names {
					score := fuzzy.Score(name, survivor)
					if score < s.cfg.MinScore {
						continue
					}
					recent, err := s.repo.RecentNotification(ctx, sub.ID, name, s.cfg.Cooldown.Std())
					if err != nil {
						s.log.Warn("notify: dedup check failed", slog.Any("error", err))
						continue
					}
					if recent {
						continue
					}
					msg := fmt.Sprintf("Player '%s' detected in your game (%.0f%% match)", survivor, score*100)
					s.irc.Send(sub.TwitchLogin, msg)
					if err := s.repo.LogNotification(ctx, sub.ID, name, score, snapID, sample.StreamerID); err != nil {
						s.log.Warn("notify: log failed", slog.Any("error", err))
					}
					s.log.Info("notify: sent",
						slog.String("to", sub.TwitchLogin),
						slog.String("detected", survivor),
						slog.Float64("score", score))
					break
				}
			}
		}
	}
}

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
		s.log.Warn("notify: steam api call failed", slog.Any("error", err))
		return nil
	}

	out := make(map[int64][]string)
	for i, sub := range subs {
		if i < len(names) {
			if nn := fuzzy.Norm(names[i]); nn != "" {
				out[sub.ID] = []string{nn}
			}
		}
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
		})
	}

	s.cache = out
	s.cacheUntil = time.Now().Add(2 * time.Minute)
	return out
}
