package entity

import "time"

// StreamSession is a reconstructed continuous "online" span of a single
// broadcaster. It is opened the first poll a stream is seen live and closed
// (EndedAt set) the first poll it is no longer present.
type StreamSession struct {
	ID             int64
	StreamerID     string
	TwitchStreamID string
	StartedAt      time.Time
	EndedAt        *time.Time
	VodID          *string
	VodCreatedAt   *time.Time
}
