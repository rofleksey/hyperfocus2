package twitch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/samber/oops"
)

type BotTokenStore struct {
	cfg          BotConfig
	mu           sync.RWMutex
	refreshMu    sync.Mutex
	accessToken  string
	refreshToken string
}

type BotConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

func NewBotTokenStore(cfg BotConfig) *BotTokenStore {
	return &BotTokenStore{cfg: cfg, refreshToken: cfg.RefreshToken}
}

func (s *BotTokenStore) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessToken
}

func (s *BotTokenStore) Refresh(ctx context.Context) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	s.mu.RLock()
	rt := s.refreshToken
	s.mu.RUnlock()

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {rt},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://id.twitch.tv/oauth2/token",
		strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return oops.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oops.Errorf("bot token refresh: status %d", resp.StatusCode)
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        []string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return oops.Wrap(err)
	}

	s.mu.Lock()
	s.accessToken = out.AccessToken
	if out.RefreshToken != "" {
		s.refreshToken = out.RefreshToken
	}
	s.mu.Unlock()
	return nil
}
