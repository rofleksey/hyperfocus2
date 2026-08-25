package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"hyperfocus/internal/config"
	"hyperfocus/internal/entity"
	"hyperfocus/internal/pkg/httputil"
	"hyperfocus/internal/pkg/steam"
	"hyperfocus/internal/pkg/twitch"
)

type SubscribeRepo interface {
	GetSubscriberByTwitch(ctx context.Context, twitchLogin string) (*entity.NotificationSubscriber, error)
	InsertSubscriber(ctx context.Context, sub entity.NotificationSubscriber) (int64, error)
	UpdateSubscriberStatus(ctx context.Context, subscriberID int64, status string) error
	DeleteSubscriber(ctx context.Context, subscriberID int64) error
	UpdateSteamNames(ctx context.Context, names map[int64]string) error
}

type SubscribeHandler struct {
	log       *slog.Logger
	repo      SubscribeRepo
	botHelix  *twitch.BotHelix
	irc       *twitch.IRCBot
	notifyCfg config.Notify
	steamCfg  config.Steam
}

func NewSubscribeHandler(log *slog.Logger, repo SubscribeRepo, botHelix *twitch.BotHelix, irc *twitch.IRCBot, notifyCfg config.Notify, steamCfg config.Steam) *SubscribeHandler {
	return &SubscribeHandler{log: log, repo: repo, botHelix: botHelix, irc: irc, notifyCfg: notifyCfg, steamCfg: steamCfg}
}

type subscribeRequest struct {
	TwitchLogin string `json:"twitch_login"`
	SteamURL    string `json:"steam_url"`
}

type subscribeResponse struct {
	Status    string `json:"status"`
	SteamName string `json:"steam_name"`
	Message   string `json:"message,omitempty"`
}

func (h *SubscribeHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	if !h.notifyCfg.IsEnabled() {
		httputil.Problem(w, http.StatusServiceUnavailable, "notifications are disabled")
		return
	}

	if r.Method == http.MethodGet {
		h.handleGet(w, r)
		return
	}
	if r.Method != http.MethodPost {
		httputil.Problem(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req subscribeRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.TwitchLogin = strings.ToLower(strings.TrimSpace(req.TwitchLogin))
	req.SteamURL = strings.TrimSpace(req.SteamURL)

	if req.TwitchLogin == "" || req.SteamURL == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "twitch_login and steam_url are required"})
		return
	}

	existing, err := h.repo.GetSubscriberByTwitch(r.Context(), req.TwitchLogin)
	if err == nil && existing != nil {
		// Fetch Steam name on the fly for the response (and persist it).
		steamName := h.fetchSteamName(r.Context(), existing)
		httputil.JSON(w, http.StatusConflict, subscribeResponse{
			Status:    existing.Status,
			SteamName: steamName,
			Message:   "subscription already exists; type !hyperfocussub in your chat if status is pending",
		})
		return
	}

	twitchUserID, _, err := h.botHelix.ResolveUser(r.Context(), req.TwitchLogin)
	if err != nil {
		h.log.Warn("subscribe: resolve twitch user failed", slog.String("login", req.TwitchLogin), slog.Any("error", err))
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "twitch user not found"})
		return
	}

	steamID, err := steam.ExtractSteamID(req.SteamURL)
	if err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid Steam URL"})
		return
	}

	if h.steamCfg.APIKey != "" {
		sc := steam.NewClient(h.steamCfg.APIKey, h.log, h.steamCfg.Retries)
		resolved, err := sc.ResolveVanity(r.Context(), steamID)
		if err == nil {
			steamID = resolved
		}
	}

	steamName, nameOK := h.resolveSteamName(r.Context(), steamID)

	sub := entity.NotificationSubscriber{
		TwitchLogin:  req.TwitchLogin,
		TwitchUserID: twitchUserID,
		SteamURL:     req.SteamURL,
		SteamID:      steamID,
	}
	if nameOK {
		sub.SteamName = steamName
	}

	id, err := h.repo.InsertSubscriber(r.Context(), sub)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httputil.JSON(w, http.StatusConflict, subscribeResponse{
				Status:  "pending",
				Message: "subscription already exists",
			})
			return
		}
		h.log.Error("subscribe: insert failed", slog.Any("error", err))
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.log.Info("subscribe: new subscriber", slog.Int64("id", id), slog.String("twitch", req.TwitchLogin), slog.String("steam_id", steamID))

	h.irc.Join(req.TwitchLogin)
	h.irc.Send(r.Context(), req.TwitchLogin, "Subscription request received! Type !hyperfocussub in your chat to verify ownership.")

	httputil.JSON(w, http.StatusCreated, subscribeResponse{
		Status:    "pending",
		SteamName: steamName,
		Message:   "type !hyperfocussub in your Twitch chat to verify",
	})
}

// resolveSteamName fetches the current persona name for a SteamID. It returns
// (name, true) on a successful API call; on failure it returns (steamID,
// false) so the caller can still display something without persisting a bogus
// name.
func (h *SubscribeHandler) resolveSteamName(ctx context.Context, steamID string) (string, bool) {
	if h.steamCfg.APIKey == "" || steamID == "" {
		return steamID, false
	}
	sc := steam.NewClient(h.steamCfg.APIKey, h.log, h.steamCfg.Retries)
	if name, err := sc.RefreshName(ctx, steamID); err == nil {
		return name, true
	}
	return steamID, false
}

// fetchSteamName resolves and persists the Steam name for an existing
// subscriber. On success the fresh name is stored in the database (so it can
// serve as a fallback later) and returned; on failure the stored name or the
// raw SteamID is returned.
func (h *SubscribeHandler) fetchSteamName(ctx context.Context, sub *entity.NotificationSubscriber) string {
	if h.steamCfg.APIKey == "" || sub.SteamID == "" {
		return sub.SteamID
	}
	sc := steam.NewClient(h.steamCfg.APIKey, h.log, h.steamCfg.Retries)
	if name, err := sc.RefreshName(ctx, sub.SteamID); err == nil {
		if err := h.repo.UpdateSteamNames(ctx, map[int64]string{sub.ID: name}); err != nil {
			h.log.Warn("subscribe: failed to store steam name", slog.Any("error", err))
		}
		return name
	}
	if sub.SteamName != "" {
		return sub.SteamName
	}
	return sub.SteamID
}

func (h *SubscribeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	twitchLogin := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("twitch")))
	if twitchLogin == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "?twitch=<login> required"})
		return
	}
	sub, err := h.repo.GetSubscriberByTwitch(r.Context(), twitchLogin)
	if err != nil || sub == nil {
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	// Deliberately exposes only the status: Steam persona names of arbitrary
	// users must not leak through this unauthenticated endpoint.
	httputil.JSON(w, http.StatusOK, subscribeResponse{Status: sub.Status})
}

// HandleIRCCommand processes !hyperfocussub / !hyperfocusunsub from the IRC bot.
func (h *SubscribeHandler) HandleIRCCommand(ctx context.Context, cmd twitch.IRCCommand) {
	switch cmd.Command {
	case "!hyperfocussub":
		sub, err := h.repo.GetSubscriberByTwitch(ctx, cmd.SenderLogin)
		if err != nil || sub == nil {
			h.log.Debug("subscribe: irc command for unknown user", slog.String("login", cmd.SenderLogin))
			return
		}
		if sub.Status == "active" {
			h.irc.Send(ctx, cmd.Channel, "You are already subscribed!")
			return
		}
		if err := h.repo.UpdateSubscriberStatus(ctx, sub.ID, "active"); err != nil {
			h.log.Error("subscribe: verify failed", slog.Any("error", err))
			return
		}
		h.irc.Send(ctx, cmd.Channel, "Verified! You'll get a heads-up when you might be playing with another streamer.")
		h.log.Info("subscribe: verified", slog.String("twitch", cmd.SenderLogin))

	case "!hyperfocusunsub":
		sub, err := h.repo.GetSubscriberByTwitch(ctx, cmd.SenderLogin)
		if err != nil || sub == nil {
			return
		}
		if err := h.repo.DeleteSubscriber(ctx, sub.ID); err != nil {
			h.log.Error("subscribe: unsub failed", slog.Any("error", err))
			return
		}
		h.irc.Part(cmd.Channel)
		h.irc.Send(ctx, cmd.Channel, "You have been unsubscribed from hyperfocus notifications.")
		h.log.Info("subscribe: unsubscribed", slog.String("twitch", cmd.SenderLogin))
	}
}
