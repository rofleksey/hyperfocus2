package entity

import "time"

// Vod is a Twitch archive video resolved from a live stream session.
type Vod struct {
	VodID           string
	StreamerID      string
	StreamID        *string
	StartedAt       time.Time
	DurationSeconds *int
	URL             string
	ThumbnailURL    string
}

// LiveStream is the projection of Twitch Helix "GET /streams" data the poll
// usecase consumes. It is a pure entity; the twitch client maps Helix types
// into it.
type LiveStream struct {
	TwitchStreamID string
	TwitchUserID   string
	Login          string
	DisplayName    string
	Title          string
	GameID         string
	ViewerCount    int
	Language       string
	StartedAt      time.Time
	ThumbnailURL   string
	Tags           []string
}

// Video is the projection of Twitch Helix "GET /videos" data.
type Video struct {
	VodID     string
	StreamID  *string
	UserID    string
	Title     string
	CreatedAt time.Time
	Duration  time.Duration
	Thumbnail string
	URL       string
}
