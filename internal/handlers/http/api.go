// Package http holds the HTTP delivery layer: handlers, routing and middleware.
// Handlers translate HTTP <-> usecase calls and map entities to JSON DTOs. They
// contain no business logic.
package http

import (
	"context"
	"log/slog"
	"net/http"
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

// StatsRepo is the read port for the /api/stats endpoints.
type StatsRepo interface {
	SnapshotStats(ctx context.Context, n int) ([]entity.SnapshotStat, error)
	VerifiedSubscriberSeries(ctx context.Context) ([]entity.SubscriberDay, error)
}

// PreviewServer exposes the on-disk directory holding preview images.
type PreviewServer interface {
	Dir() string
}

// Pinger verifies the database connection (pgxpool.Pool satisfies it).
type Pinger interface {
	Ping(ctx context.Context) error
}

// PublicConfig is the subset of server config exposed via GET /api/config so
// the SPA never hardcodes operational facts (retention window, feature
// flags).
type PublicConfig struct {
	RetentionHours int
	NotifyEnabled  bool
}

// Deps holds the collaborators wired into the HTTP layer.
type Deps struct {
	Logger    *slog.Logger
	Moments   *moments.Service
	Streamers StreamerQuery
	Previews  PreviewServer
	StatsRepo StatsRepo
	DB        Pinger
	Version   string
	Subscribe *SubscribeHandler
	Public    PublicConfig
}

// API groups handlers over shared dependencies.
type API struct {
	log       *slog.Logger
	moments   *moments.Service
	streamers StreamerQuery
	previews  PreviewServer
	statsRepo StatsRepo
	db        Pinger
	version   string
	subscribe *SubscribeHandler
	public    PublicConfig
}

// NewAPI builds an API from deps.
func NewAPI(d Deps) *API {
	return &API{
		log:       d.Logger,
		moments:   d.Moments,
		streamers: d.Streamers,
		previews:  d.Previews,
		statsRepo: d.StatsRepo,
		db:        d.DB,
		version:   d.Version,
		subscribe: d.Subscribe,
		public:    d.Public,
	}
}

func (a *API) Subscribe(w http.ResponseWriter, r *http.Request) {
	a.subscribe.HandleSubscribe(w, r)
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
	StreamerID      string    `json:"streamer_id"`
	Login           string    `json:"login"`
	DisplayName     string    `json:"display_name"`
	ProfileImageURL string    `json:"profile_image_url,omitempty"`
	ViewerCount     int       `json:"viewer_count"`
	Title           string    `json:"title"`
	Language        string    `json:"language,omitempty"`
	Tags            []string  `json:"tags"`
	StartedAt       time.Time `json:"started_at"`
	PreviewURL      string    `json:"preview_url,omitempty"`
	ThumbURL        string    `json:"thumb_url,omitempty"`
	TwitchURL       string    `json:"twitch_url,omitempty"`
	SurvivorNames   []string  `json:"survivor_names"`
	FuzzyScore      *float64  `json:"fuzzy_score,omitempty"`
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
		StreamerID:      d.StreamerID,
		Login:           d.Login,
		DisplayName:     d.DisplayName,
		ProfileImageURL: d.ProfileImageURL,
		ViewerCount:     d.ViewerCount,
		Title:           d.Title,
		Language:        d.Language,
		Tags:            d.Tags,
		StartedAt:       d.StartedAt,
		SurvivorNames:   d.SurvivorNames,
		FuzzyScore:      d.FuzzyScore,
		TwitchURL:       "https://www.twitch.tv/" + d.Login,
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
	return out
}
