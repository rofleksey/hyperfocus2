package http

import (
	"errors"
	"net/http"
	"strconv"

	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/httputil"
)

type streamerDTO struct {
	TwitchUserID    string `json:"twitch_user_id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
}

type sessionDTO struct {
	ID             int64  `json:"id"`
	TwitchStreamID string `json:"twitch_stream_id"`
	StartedAt      string `json:"started_at"`
	EndedAt        string `json:"ended_at,omitempty"`
	VodURL         string `json:"vod_url,omitempty"`
}

// Streamers handles GET /api/streamers?q=&limit=.
func (a *API) Streamers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	list, err := a.streamers.ListStreamers(r.Context(), q.Get("q"), limit)
	if err != nil {
		a.log.Error("list streamers failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}
	out := make([]streamerDTO, 0, len(list))
	for _, s := range list {
		out = append(out, toStreamerDTO(s))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": out})
}

// Streamer handles GET /api/streamers/{id}.
func (a *API) Streamer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.Problem(w, http.StatusBadRequest, "missing id")
		return
	}

	s, err := a.streamers.GetStreamer(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			httputil.Problem(w, http.StatusNotFound, "streamer not found")
			return
		}
		a.log.Error("get streamer failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}

	sessions, err := a.streamers.ListSessionsForStreamer(r.Context(), id, 50)
	if err != nil {
		a.log.Error("list sessions failed", "error", err)
		httputil.Problem(w, http.StatusInternalServerError, "query failed")
		return
	}

	sessOut := make([]sessionDTO, 0, len(sessions))
	for _, s := range sessions {
		sessOut = append(sessOut, toSessionDTO(s))
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"streamer": toStreamerDTO(s),
		"sessions": sessOut,
	})
}

func toStreamerDTO(s entity.Streamer) streamerDTO {
	return streamerDTO{
		TwitchUserID:    s.TwitchUserID,
		Login:           s.Login,
		DisplayName:     s.DisplayName,
		ProfileImageURL: s.ProfileImageURL,
	}
}

func toSessionDTO(s entity.SessionDetail) sessionDTO {
	out := sessionDTO{
		ID:             s.ID,
		TwitchStreamID: s.TwitchStreamID,
		StartedAt:      s.StartedAt.UTC().Format(timeRFC3339),
	}
	if s.EndedAt != nil {
		out.EndedAt = s.EndedAt.UTC().Format(timeRFC3339)
	}
	if s.VodID != nil && *s.VodID != "" {
		out.VodURL = "https://www.twitch.tv/videos/" + *s.VodID
	}
	return out
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
