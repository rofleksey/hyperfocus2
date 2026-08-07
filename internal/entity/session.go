package entity

import "time"

// StreamSession is a reconstructed continuous "online" span of a single
// broadcaster. It is opened the first poll a stream is seen live and closed
// (EndedAt set) the first poll it is no longer present. The VOD fields are
// resolved live via Get Videos (matching stream_id) by the resolvevod usecase.
type StreamSession struct {
	ID             int64
	StreamerID     string
	TwitchStreamID string
	StartedAt      time.Time
	EndedAt        *time.Time
	VodID          *string
	VodCreatedAt   *time.Time
}

// IsOpen reports whether the session is still considered live at the given time.
func (s StreamSession) IsOpen() bool { return s.EndedAt == nil }
