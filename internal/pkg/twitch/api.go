package twitch

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/nicklaw5/helix/v2"
	"github.com/samber/oops"
)

type BotHelix struct {
	helix   *helix.Client
	mu      sync.Mutex
	log     *slog.Logger
	token   *BotTokenStore
	botUserID string
}

func NewBotHelix(cfg BotConfig, log *slog.Logger) (*BotHelix, error) {
	token := NewBotTokenStore(cfg)
	hc, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		HTTPClient:   &http.Client{Timeout: 15 * time.Second},
	})
	if err != nil {
		return nil, oops.Wrap(err)
	}

	b := &BotHelix{helix: hc, log: log, token: token}
	if err := token.Refresh(context.Background()); err != nil {
		return nil, oops.Wrap(err)
	}

	hc.SetUserAccessToken(token.Get())

	resp, err := hc.GetUsers(&helix.UsersParams{})
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if resp.StatusCode != http.StatusOK || len(resp.Data.Users) == 0 {
		return nil, oops.Errorf("validate bot token: status %d", resp.StatusCode)
	}
	b.botUserID = resp.Data.Users[0].ID
	log.Info("twitch_bot: authenticated", slog.String("user_id", b.botUserID), slog.String("login", resp.Data.Users[0].Login))
	return b, nil
}

func (b *BotHelix) BotUserID() string { return b.botUserID }

func (b *BotHelix) UserAccessToken() string { return b.token.Get() }

func (b *BotHelix) RefreshLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := b.token.Refresh(ctx); err != nil {
				b.log.Error("twitch_bot: token refresh failed", slog.Any("error", err))
			}
		}
	}
}

func (b *BotHelix) setToken() {
	b.mu.Lock()
	b.helix.SetUserAccessToken(b.token.Get())
	b.mu.Unlock()
}

func (b *BotHelix) ResolveUser(ctx context.Context, login string) (id string, displayName string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.helix.SetUserAccessToken(b.token.Get())
	resp, err := b.helix.GetUsers(&helix.UsersParams{Logins: []string{login}})
	if err != nil {
		return "", "", oops.Wrap(err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", oops.Errorf("resolve user: status %d", resp.StatusCode)
	}
	if len(resp.Data.Users) == 0 {
		return "", "", oops.Errorf("twitch user not found: %s", login)
	}
	return resp.Data.Users[0].ID, resp.Data.Users[0].DisplayName, nil
}

func (b *BotHelix) SendChatMessage(ctx context.Context, broadcasterID, message string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.helix.SetUserAccessToken(b.token.Get())
	resp, err := b.helix.SendChatMessage(&helix.SendChatMessageParams{
		BroadcasterID: broadcasterID,
		SenderID:      b.botUserID,
		Message:       message,
	})
	if err != nil {
		return oops.Wrap(err)
	}
	if resp.StatusCode != http.StatusOK {
		return oops.Errorf("send chat: status %d: %s", resp.StatusCode, resp.ErrorMessage)
	}
	return nil
}
