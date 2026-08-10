package entity

import "time"

// Snapshot is a single recorded "moment": one poll cycle of the live directory.
type Snapshot struct {
	ID              int64
	TakenAt         time.Time
	Source          string
	StreamCount     int
	DurationSeconds float64
}
