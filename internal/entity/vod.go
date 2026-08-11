package entity

import "time"

// LiveStream is the projection of Twitch Helix "GET /streams" data the poll
// usecase consumes. It is a pure entity; the twitch client maps Helix types
// into it.
type LiveStream struct {
	TwitchStreamID string
	TwitchUserID   string
	Login          string
	DisplayName    string
	Title          string
	ViewerCount    int
	Language       string
	StartedAt      time.Time
	ThumbnailURL   string
	Tags           []string
}
