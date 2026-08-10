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
	apiKey string
	log    *slog.Logger
	client *http.Client
}

func NewClient(apiKey string, log *slog.Logger) *Client {
	return &Client{
		apiKey: apiKey,
		log:    log,
		client: &http.Client{Timeout: 15 * time.Second},
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
	if len(parts) == 0 {
		return "", oops.Errorf("invalid Steam profile URL")
	}

	switch parts[0] {
	case "profiles":
		if len(parts) < 2 {
			return "", oops.Errorf("invalid Steam profiles URL")
		}
		return parts[1], nil
	case "id":
		if len(parts) < 2 {
			return "", oops.Errorf("invalid Steam id URL")
		}
		return resolveVanity(u.String(), parts[1])
	default:
		return "", oops.Errorf("unrecognised Steam URL path: %s", parts[0])
	}
}

func resolveVanity(profileURL, vanity string) (string, error) {
	return "", oops.Errorf("vanity URL resolution requires Steam API key; use /profiles/<steamID64> format")
}

// GetPlayerSummaries fetches persona names for up to 100 SteamIDs.
func (c *Client) GetPlayerSummaries(ctx context.Context, steamIDs []string) ([]PlayerSummary, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}
	if len(steamIDs) > 100 {
		steamIDs = steamIDs[:100]
	}
	ids := strings.Join(steamIDs, ",")
	reqURL := fmt.Sprintf("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/?key=%s&steamids=%s", c.apiKey, ids)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	resp, err := c.client.Do(req)
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
}

// RefreshName fetches the current persona name for a single SteamID.
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
