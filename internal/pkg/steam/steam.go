package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/samber/oops"
)

type Client struct {
	apiKey  string
	retries int
	log     *slog.Logger
	client  *http.Client
}

func NewClient(apiKey string, log *slog.Logger, retries int) *Client {
	return &Client{
		apiKey:  apiKey,
		retries: retries,
		log:     log,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type PlayerSummary struct {
	SteamID     string `json:"steamid"`
	PersonaName string `json:"personaname"`
	ProfileURL  string `json:"profileurl"`
}

// ExtractSteamID parses a steamcommunity.com URL and returns the SteamID64.
func ExtractSteamID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", oops.Wrap(err)
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "steamcommunity.com") {
		return "", oops.Errorf("not a steamcommunity.com URL")
	}

	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", oops.Errorf("invalid Steam profile URL")
	}

	switch parts[0] {
	case "profiles":
		return parts[1], nil
	case "id":
		return parts[1], nil // vanity — resolved via API at refresh time
	default:
		return "", oops.Errorf("unrecognised Steam URL path: %s", parts[0])
	}
}

// withRetry runs fn up to attempts+1 times (attempts = number of retries after
// the first try), waiting 1s between attempts. It returns the first success or
// the last error. The context is honoured both inside fn and between retries.
func withRetry[T any](ctx context.Context, attempts int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 0; attempt <= attempts; attempt++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return zero, lastErr
}

func (c *Client) ResolveVanity(ctx context.Context, vanity string) (string, error) {
	return withRetry(ctx, c.retries, func() (string, error) {
		reqURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?key=%s&vanityurl=%s", c.apiKey, vanity)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		resp, err := c.client.Do(req)
		if err != nil {
			return "", oops.Wrap(err)
		}
		defer resp.Body.Close()

		var out struct {
			Response struct {
				SteamID string `json:"steamid"`
				Success int    `json:"success"`
			} `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "", oops.Wrap(err)
		}
		if out.Response.Success != 1 {
			return "", oops.Errorf("vanity not found: %s", vanity)
		}
		return out.Response.SteamID, nil
	})
}

// GetPlayerSummaries fetches persona names for up to 100 Steam IDs, retrying
// on transient failures (timeouts, non-200s, decode errors).
func GetPlayerSummaries(ctx context.Context, apiKey string, steamIDs []string, retries int) ([]PlayerSummary, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}
	if len(steamIDs) > 100 {
		steamIDs = steamIDs[:100]
	}
	ids := strings.Join(steamIDs, ",")
	reqURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", apiKey, ids)

	return withRetry(ctx, retries, func() ([]PlayerSummary, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, oops.Wrap(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, oops.Errorf("steam api: status %d", resp.StatusCode)
		}

		var out struct {
			Response struct {
				Players []struct {
					SteamID     string `json:"steamid"`
					PersonaName string `json:"personaname"`
					ProfileURL  string `json:"profileurl"`
				} `json:"players"`
			} `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, oops.Wrap(err)
		}

		players := make([]PlayerSummary, len(out.Response.Players))
		for i, p := range out.Response.Players {
			players[i] = PlayerSummary{
				SteamID:     p.SteamID,
				PersonaName: p.PersonaName,
				ProfileURL:  p.ProfileURL,
			}
		}
		return players, nil
	})
}

func (c *Client) RefreshName(ctx context.Context, steamID string) (string, error) {
	players, err := c.GetPlayerSummaries(ctx, []string{steamID})
	if err != nil {
		return "", err
	}
	if len(players) == 0 {
		return "", oops.Errorf("steam player not found: %s", steamID)
	}
	return players[0].PersonaName, nil
}

func (c *Client) GetPlayerSummaries(ctx context.Context, steamIDs []string) ([]PlayerSummary, error) {
	return GetPlayerSummaries(ctx, c.apiKey, steamIDs, c.retries)
}
