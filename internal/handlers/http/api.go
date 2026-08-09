// Package http holds the HTTP delivery layer: handlers, routing and middleware.
// Handlers translate HTTP <-> usecase calls and map entities to JSON DTOs. They
// contain no business logic.
package http

import (
	"context"
	"log/slog"
	"time"

	"hyperfocus/internal/entity"
	"hyperfocus/internal/usecases/moments"
)

// StreamerQuery is the read port for streamer endpoints.
type StreamerQuery interface {
	ListStreamers(ctx context.Context, q string, limit int) ([]entity.Streamer, error)
	GetStreamer(ctx context.Context, id string) (entity.Streamer, error)
	ListSessionsForStreamer(ctx context.Context, streamerID string, limit int) ([]entity.SessionDetail, error)
}

// PreviewServer exposes the on-disk directory holding preview images.
type PreviewServer interface {
	Dir() string
}

// VodQuery is the read port for VOD endpoints.
type VodQuery interface {
	GetVod(ctx context.Context, vodID string) (entity.Vod, error)
}

// Deps holds the collaborators wired into the HTTP layer.
type Deps struct {
	Logger    *slog.Logger
	Moments   *moments.Service
	Streamers StreamerQuery
	Vods      VodQuery
	Previews  PreviewServer
	Version   string
}

// API groups handlers over shared dependencies.
type API struct {
	log       *slog.Logger
	moments   *moments.Service
	streamers StreamerQuery
	vods      VodQuery
	previews  PreviewServer
	version   string
}

// NewAPI builds an API from deps.
func NewAPI(d Deps) *API {
	return &API{log: d.Logger, moments: d.Moments, streamers: d.Streamers, vods: d.Vods, previews: d.Previews, version: d.Version}
}

// snapshotDTO is the JSON shape for a snapshot.
type snapshotDTO struct {
	ID          int64     `json:"id"`
	TakenAt     time.Time `json:"taken_at"`
	Source      string    `json:"source"`
	StreamCount int       `json:"stream_count"`
}

// streamDTO is the per-stream JSON shape inside a moment response.
type streamDTO struct {
	StreamerID       string    `json:"streamer_id"`
	Login            string    `json:"login"`
	DisplayName      string    `json:"display_name"`
	ProfileImageURL  string    `json:"profile_image_url,omitempty"`
	ViewerCount      int       `json:"viewer_count"`
	Title            string    `json:"title"`
	Language         string    `json:"language,omitempty"`
	Tags             []string  `json:"tags"`
	StartedAt        time.Time `json:"started_at"`
	VodOffsetSeconds *int      `json:"vod_offset_seconds,omitempty"`
	PreviewURL       string    `json:"preview_url,omitempty"`
	ThumbURL         string    `json:"thumb_url,omitempty"`
	VodURL           string    `json:"vod_url,omitempty"`
	TwitchURL        string    `json:"twitch_url,omitempty"`
	SurvivorNames    []string  `json:"survivor_names"`
	FuzzyScore       *float64  `json:"fuzzy_score,omitempty"`
}

// momentResponse is the answer to "who was online at T".
type momentResponse struct {
	RequestedAt time.Time    `json:"requested_at"`
	HasData     bool         `json:"has_data"`
	Snapshot    *snapshotDTO `json:"snapshot,omitempty"`
	Streams     []streamDTO  `json:"streams"`
}

func toSnapshotDTO(s entity.Snapshot) *snapshotDTO {
	return &snapshotDTO{
		ID:          s.ID,
		TakenAt:     s.TakenAt,
		Source:      s.Source,
		StreamCount: s.StreamCount,
	}
}

func toStreamDTO(d entity.SampleDetail) streamDTO {
	out := streamDTO{
		StreamerID:       d.StreamerID,
		Login:            d.Login,
		DisplayName:      d.DisplayName,
		ProfileImageURL:  d.ProfileImageURL,
		ViewerCount:      d.ViewerCount,
		Title:            d.Title,
		Language:         d.Language,
		Tags:             d.Tags,
		StartedAt:        d.StartedAt,
		VodOffsetSeconds: d.VodOffsetSeconds,
		SurvivorNames:    d.SurvivorNames,
		FuzzyScore:       d.FuzzyScore,
		TwitchURL:        "https://www.twitch.tv/" + d.Login,
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if out.SurvivorNames == nil {
		out.SurvivorNames = []string{}
	}
	if d.PreviewFilename != nil && *d.PreviewFilename != "" {
		out.PreviewURL = "/previews/" + *d.PreviewFilename
	}
	if d.ThumbFilename != nil && *d.ThumbFilename != "" {
		out.ThumbURL = "/previews/thumbs/" + *d.ThumbFilename
	}
	if d.VodID != nil && *d.VodID != "" {
		out.VodURL = "https://www.twitch.tv/videos/" + *d.VodID + "?t=" + formatOffset(d.VodOffsetSeconds)
	}
	return out
}

// formatOffset turns a nullable second count into Twitch's "?t=1h2m3s" suffix.
func formatOffset(seconds *int) string {
	if seconds == nil {
		return "0s"
	}
	s := *seconds
	if s < 0 {
		s = 0
	}
	h := s / 3600
	s %= 3600
	m := s / 60
	s %= 60
	out := ""
	if h > 0 {
		out += itoa(h) + "h"
	}
	if m > 0 || h > 0 {
		out += itoa(m) + "m"
	}
	out += itoa(s) + "s"
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for n > 0 {
		pos--
		b[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(b[pos:])
}
