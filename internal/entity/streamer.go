package entity

import "time"

// Streamer is the cached channel/broadcaster metadata for a Twitch user.
type Streamer struct {
	TwitchUserID    string
	Login           string
	DisplayName     string
	ProfileImageURL string
	LastSeen        time.Time
}
