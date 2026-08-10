package entity

import "time"

type SnapshotStat struct {
	ID              int64     `json:"id"`
	TakenAt         time.Time `json:"taken_at"`
	StreamCount     int       `json:"stream_count"`
	DurationSeconds float64   `json:"duration_seconds"`
	DiskUsageBytes  int64     `json:"disk_usage_bytes"`
	PreviewOK       int       `json:"preview_ok"`
	OCROK           int       `json:"ocr_ok"`
	Total           int       `json:"total"`
}
