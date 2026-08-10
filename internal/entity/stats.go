package entity

import "time"

// SnapshotStat is a single data-point returned by stats queries.
type SnapshotStat struct {
	ID          int64     `json:"id"`
	TakenAt     time.Time `json:"taken_at"`
	StreamCount int       `json:"stream_count"`
	PreviewOK   int       `json:"preview_ok"`
	OCROK       int       `json:"ocr_ok"`
	Total       int       `json:"total"`
}