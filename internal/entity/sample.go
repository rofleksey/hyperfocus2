package entity

import "time"

// StreamSample is the per-moment, per-stream record. It carries its own
// filterable/sortable data (viewers, title, language, tags) so any past moment
// can be reconstructed and queried independently.
type StreamSample struct {
	SnapshotID       int64
	SessionID        int64
	StreamerID       string
	ViewerCount      int
	Title            string
	Language         string
	Tags             []string
	StartedAt        time.Time
	VodOffsetSeconds *int
	PreviewFilename  *string
	SurvivorNames    []string
}
