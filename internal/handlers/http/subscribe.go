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
	InsertSubscriber(ctx context.Context, sub entity.NotificationSubscriber, names []string) (int64, error)
	UpdateSubscriberStatus(ctx context.Context, subscriberID int64, status string) error
	DeleteSubscriber(ctx context.Context, subscriberID int64) error
}

type SubscribeHandler struct {
	log        *slog.Logger
	repo       SubscribeRepo
	botHelix   *twitch.BotHelix
	irc        *twitch.IRCBot
	notifyCfg  config.Notify
	steamKey   string
}

func NewSubscribeHandler(log *slog.Logger, repo SubscribeRepo, botHelix *twitch.BotHelix, irc *twitch.IRCBot, notifyCfg config.Notify, steamKey string) *SubscribeHandler {
	return &SubscribeHandler{log: log, repo: repo, botHelix: botHelix, irc: irc, notifyCfg: notifyCfg, steamKey: steamKey}
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
		http.Error(w, `{"error":"notifications are disabled"}`, http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodGet {
		h.handleGet(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		h.handleDelete(w, r)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
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
		httputil.JSON(w, http.StatusConflict, subscribeResponse{
			Status:    existing.Status,
			SteamName: existing.SteamName,
			Message:   "subscription already exists; type !hyperfocussub in your chat if status is pending",
		})
		return
	}

	twitchUserID, twitchDisplayName, err := h.botHelix.ResolveUser(r.Context(), req.TwitchLogin)
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

	if h.steamKey != "" {
		sc := steam.NewClient(h.steamKey, h.log)
		resolved, err := sc.ResolveVanity(r.Context(), steamID)
		if err == nil {
			steamID = resolved
		}
	}

	var steamName string
	if h.steamKey != "" {
		sc := steam.NewClient(h.steamKey, h.log)
		if name, err := sc.RefreshName(r.Context(), steamID); err == nil {
			steamName = name
		} else {
			h.log.Warn("subscribe: steam name fetch failed", slog.Any("error", err))
		}
	}
	if steamName == "" {
		steamName = twitchDisplayName
	}

	sub := entity.NotificationSubscriber{
		TwitchLogin:  req.TwitchLogin,
		TwitchUserID: twitchUserID,
		SteamURL:     req.SteamURL,
		SteamID:      steamID,
		SteamName:    steamName,
	}

	id, err := h.repo.InsertSubscriber(r.Context(), sub, nil)
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
	h.irc.Send(req.TwitchLogin, "Subscription request received! Type !hyperfocussub in your chat to verify ownership.")

	httputil.JSON(w, http.StatusCreated, subscribeResponse{
		Status:    "pending",
		SteamName: steamName,
		Message:   "type !hyperfocussub in your Twitch chat to verify",
	})
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
	httputil.JSON(w, http.StatusOK, subscribeResponse{Status: sub.Status, SteamName: sub.SteamName})
}

func (h *SubscribeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TwitchLogin string `json:"twitch_login"`
	}
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.TwitchLogin = strings.ToLower(strings.TrimSpace(req.TwitchLogin))
	if req.TwitchLogin == "" {
		httputil.JSON(w, http.StatusBadRequest, map[string]string{"error": "twitch_login required"})
		return
	}
	sub, err := h.repo.GetSubscriberByTwitch(r.Context(), req.TwitchLogin)
	if err != nil || sub == nil {
		httputil.JSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err := h.repo.DeleteSubscriber(r.Context(), sub.ID); err != nil {
		httputil.JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.irc.Part(req.TwitchLogin)
	h.irc.Send(req.TwitchLogin, "You have been unsubscribed from hyperfocus notifications.")
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "unsubscribed"})
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
			h.irc.Send(cmd.Channel, "You are already subscribed!")
			return
		}
		if err := h.repo.UpdateSubscriberStatus(ctx, sub.ID, "active"); err != nil {
			h.log.Error("subscribe: verify failed", slog.Any("error", err))
			return
		}
		h.irc.Send(cmd.Channel, "Verified! You'll be notified when other players are detected in your DBD games.")
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
		h.irc.Send(cmd.Channel, "You have been unsubscribed from hyperfocus notifications.")
		h.log.Info("subscribe: unsubscribed", slog.String("twitch", cmd.SenderLogin))
	}
}

