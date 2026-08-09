// Package twitch is the outbound adapter for the Twitch Helix API. It owns
// app-access-token lifecycle and exposes a small, pure-entity-facing surface.
package twitch

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nicklaw5/helix/v2"
	"github.com/samber/oops"
	"golang.org/x/time/rate"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
)

// Client wraps a Helix client with app-access-token management and a rate
// limiter at 8 req/s (480/min), well under Twitch's 800 req/min limit.
type Client struct {
	cfg     config.Twitch
	helix   *helix.Client
	log     *slog.Logger
	mu      sync.Mutex
	token   string
	expires time.Time
	limiter *rate.Limiter
}

// New creates and validates a Client by obtaining its first app access token.
func New(cfg config.Twitch, log *slog.Logger) (*Client, error) {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	hc, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		HTTPClient:   httpClient,
	})
	if err != nil {
		return nil, oops.Wrap(err)
	}

	c := &Client{
		cfg:     cfg,
		helix:   hc,
		log:     log,
		limiter: rate.NewLimiter(rate.Limit(8), 8),
	}
	if err := c.refresh(); err != nil {
		c.log.Error("twitch: initial token refresh failed", slog.Any("error", err))
		return nil, err
	}
	c.log.Info("twitch: app access token obtained")
	return c, nil
}

// waitLimiter blocks until a rate-limit token is available or ctx is done.
func (c *Client) waitLimiter(ctx context.Context) error {
	return c.limiter.Wait(ctx)
}

// runRefreshLoop periodically refreshes the app access token until ctx is done.
func (c *Client) RunRefreshLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.refresh(); err != nil {
				c.log.Error("twitch: token refresh failed", slog.Any("error", err))
			} else {
				c.log.Debug("twitch: token refreshed")
			}
		}
	}
}

func (c *Client) refresh() error {
	resp, err := c.helix.RequestAppAccessToken([]string{})
	if err != nil {
		return oops.Wrap(err)
	}
	if resp.StatusCode != http.StatusOK {
		return oops.Errorf("app access token: status %d: %s", resp.StatusCode, resp.ErrorMessage)
	}
	c.mu.Lock()
	c.token = resp.Data.AccessToken
	c.expires = time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)
	c.helix.SetAppAccessToken(resp.Data.AccessToken)
	c.mu.Unlock()
	return nil
}

// ensureToken refreshes the token if it is missing or near expiry.
func (c *Client) ensureToken() error {
	c.mu.Lock()
	exp := c.expires
	c.mu.Unlock()
	if time.Now().Add(5 * time.Minute).Before(exp) {
		return nil
	}
	c.log.Debug("twitch: token near expiry, refreshing")
	if err := c.refresh(); err != nil {
		c.log.Error("twitch: token refresh failed during ensure", slog.Any("error", err))
		return err
	}
	return nil
}

// GetLiveGameStreams returns all currently-live streams for the configured game,
// paging through the entire result set.
func (c *Client) GetLiveGameStreams(ctx context.Context) ([]entity.LiveStream, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	c.log.Debug("twitch: fetching live streams")
	pageSize := 100
	var out []entity.LiveStream
	var after string
	for {
		if err := c.waitLimiter(ctx); err != nil {
			return nil, err
		}
		resp, err := c.helix.GetStreams(&helix.StreamsParams{
			First:   pageSize,
			After:   after,
			GameIDs: []string{c.cfg.GameID},
			Type:    "live",
		})
		if err != nil {
			c.log.Error("twitch: get streams api error",
				slog.Any("error", err),
				slog.String("after", after))
			return nil, oops.Wrap(err)
		}
		if resp.StatusCode != http.StatusOK {
			c.log.Error("twitch: get streams non-ok status",
				slog.Int("status", resp.StatusCode),
				slog.String("message", resp.ErrorMessage))
			return nil, oops.Errorf("get streams: status %d: %s", resp.StatusCode, resp.ErrorMessage)
		}
		for _, s := range resp.Data.Streams {
			out = append(out, mapStream(s))
		}
		after = resp.Data.Pagination.Cursor
		if after == "" {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	c.log.Debug("twitch: live streams fetched", slog.Int("count", len(out)))
	return out, nil
}

// GetVideosByUser returns the most recent archived videos for a broadcaster.
// Used to resolve a live session's vod_id (matching stream_id).
func (c *Client) GetVideosByUser(ctx context.Context, userID string) ([]entity.Video, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	c.log.Debug("twitch: fetching videos", slog.String("user_id", userID))

	if err := c.waitLimiter(ctx); err != nil {
		return nil, err
	}
	resp, err := c.helix.GetVideos(&helix.VideosParams{
		UserID: userID,
		Type:   "archive",
		First:  20,
	})
	if err != nil {
		c.log.Error("twitch: get videos api error",
			slog.Any("error", err),
			slog.String("user_id", userID))
		return nil, oops.Wrap(err)
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Error("twitch: get videos non-ok status",
			slog.Int("status", resp.StatusCode),
			slog.String("message", resp.ErrorMessage),
			slog.String("user_id", userID))
		return nil, oops.Errorf("get videos: status %d: %s", resp.StatusCode, resp.ErrorMessage)
	}

	out := make([]entity.Video, 0, len(resp.Data.Videos))
	for _, v := range resp.Data.Videos {
		out = append(out, mapVideo(v, c.log))
	}
	c.log.Debug("twitch: videos fetched",
		slog.String("user_id", userID),
		slog.Int("count", len(out)))
	return out, nil
}

func mapStream(s helix.Stream) entity.LiveStream {
	return entity.LiveStream{
		TwitchStreamID: s.ID,
		TwitchUserID:   s.UserID,
		Login:          s.UserLogin,
		DisplayName:    s.UserName,
		Title:          s.Title,
		GameID:         s.GameID,
		ViewerCount:    s.ViewerCount,
		Language:       s.Language,
		StartedAt:      utc(s.StartedAt),
		ThumbnailURL:   s.ThumbnailURL,
		Tags:           append([]string(nil), s.Tags...),
	}
}

func mapVideo(v helix.Video, log *slog.Logger) entity.Video {
	var streamID *string
	if strings.TrimSpace(v.StreamID) != "" {
		sid := v.StreamID
		streamID = &sid
	}
	dur := parseTwitchDuration(v.Duration)
	t := parseTime(v.CreatedAt)
	if t.IsZero() && v.CreatedAt != "" {
		log.Warn("twitch: failed to parse video created_at",
			slog.String("vod_id", v.ID),
			slog.String("raw", v.CreatedAt))
	}
	vid := entity.Video{
		VodID:     v.ID,
		StreamID:  streamID,
		UserID:    v.UserID,
		Title:     v.Title,
		CreatedAt: t,
		Duration:  dur,
		Thumbnail: v.ThumbnailURL,
		URL:       v.URL,
	}
	return vid
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// utc returns the UTC-normalized time, preserving the zero value.
func utc(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC()
}

// parseTwitchDuration parses Twitch's "1h2m3s" / "30m10s" / "45s" format.
func parseTwitchDuration(s string) time.Duration {
	var h, m, sec int
	parts := strings.NewReader(s)
	read := func(suffix byte) (int, bool) {
		var num strings.Builder
		for {
			ch, _, err := parts.ReadRune()
			if err != nil {
				return 0, false
			}
			if ch == rune(suffix) {
				n, err := strconv.Atoi(num.String())
				if err != nil {
					return 0, false
				}
				return n, true
			}
			if ch < '0' || ch > '9' {
				_ = parts.UnreadRune()
				return 0, false
			}
			num.WriteRune(ch)
		}
	}
	if v, ok := read('h'); ok {
		h = v
	}
	if v, ok := read('m'); ok {
		m = v
	}
	if v, ok := read('s'); ok {
		sec = v
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}
