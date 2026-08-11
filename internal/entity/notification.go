package entity

import "time"

type NotificationSubscriber struct {
	ID           int64
	TwitchLogin  string
	TwitchUserID string
	SteamURL     string
	SteamID      string
	Status       string
	CreatedAt    time.Time
	VerifiedAt   *time.Time
}

type NotificationLog struct {
	ID               int64
	SubscriberID     int64
	DetectedName     string
	MatchScore       float64
	SnapshotID       int64
	SourceStreamerID string
	SentAt           time.Time
}
