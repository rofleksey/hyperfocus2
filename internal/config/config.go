// Package config loads application configuration from a YAML file and validates
// it. Defaults are applied centrally for any unset values.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration object.
type Config struct {
	Service   Service   `yaml:"service"`
	DB        DB        `yaml:"db"`
	Twitch    Twitch    `yaml:"twitch"`
	TwitchBot TwitchBot `yaml:"twitch_bot"`
	Poll      Poll      `yaml:"poll"`
	Prune     Prune     `yaml:"prune"`
	Storage   Storage   `yaml:"storage"`
	OCR       OCR       `yaml:"ocr"`
	Notify    Notify    `yaml:"notify"`
	Steam     Steam     `yaml:"steam"`
	Log       Log       `yaml:"log"`
}

type Service struct {
	Name     string `yaml:"name"      validate:"required"`
	HTTPAddr string `yaml:"http_addr" validate:"required"`
}

type DB struct {
	Host     string `yaml:"host"     validate:"required"`
	Port     int    `yaml:"port"     validate:"required"`
	User     string `yaml:"user"     validate:"required"`
	Password string `yaml:"password"`
	Database string `yaml:"database" validate:"required"`
	SSLMode  string `yaml:"ssl_mode"`
	MaxConns int32  `yaml:"max_conns"`
}

type Twitch struct {
	ClientID     string `yaml:"client_id"     validate:"required"`
	ClientSecret string `yaml:"client_secret" validate:"required"`
	GameID       string `yaml:"game_id"`
}

type Poll struct {
	PageSize         int      `yaml:"page_size"`
	PageDelay        Duration `yaml:"page_delay"`
	PreviewWorkers   int      `yaml:"preview_workers"`
	PreviewTimeout   Duration `yaml:"preview_timeout"`
	FetchMaxAttempts int      `yaml:"fetch_max_attempts"`
	FetchDelay       Duration `yaml:"fetch_delay"`
}

type Prune struct {
	Interval Duration `yaml:"interval"`
	Hours    int      `yaml:"hours"`
}

type Storage struct {
	DataDir string `yaml:"data_dir"`
}

// OCR configures the survivor-name OCR client. OCR runs in an external
// microservice (hyperfocus2-ocr) that keeps the RapidOCR / PaddleOCR-ONNX
// model resident, so each preview is POSTed to it individually — there are no
// batches and no per-cycle model load. api_url is the base URL of that service
// (e.g. "http://localhost:8081" or a Traefik frontend fronting N replicas).
// workers is the number of Go goroutines issuing concurrent OCR requests.
// Enabled is a pointer so an explicit `enabled: false` in YAML is honored while
// an omitted field still defaults to enabled.
type OCR struct {
	Enabled *bool    `yaml:"enabled"`
	APIURL  string   `yaml:"api_url"`
	Workers int      `yaml:"workers"`
	Timeout Duration `yaml:"timeout"`
}

// IsEnabled reports whether OCR is enabled, defaulting to true when the field is
// unset.
func (o OCR) IsEnabled() bool { return o.Enabled == nil || *o.Enabled }

type Notify struct {
	Enabled  *bool    `yaml:"enabled"`
	MinScore float64  `yaml:"min_score"`
	Cooldown Duration `yaml:"cooldown"`
	Workers  int      `yaml:"workers"`
}

func (n Notify) IsEnabled() bool { return n.Enabled == nil || *n.Enabled }

type TwitchBot struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RefreshToken string `yaml:"refresh_token"`
}

func (t TwitchBot) ClientIDor(parent string) string {
	if t.ClientID != "" {
		return t.ClientID
	}
	return parent
}

func (t TwitchBot) ClientSecretOr(parent string) string {
	if t.ClientSecret != "" {
		return t.ClientSecret
	}
	return parent
}

type Steam struct {
	APIKey       string   `yaml:"api_key"`
	RefreshEvery Duration `yaml:"refresh_every"`
	Retries      int      `yaml:"retries"`
}

type Log struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load reads the YAML file (if path is non-empty), applies defaults, then
// validates. A non-existent path is an error.
func Load(path string) (*Config, error) {
	cfg := &Config{Log: Log{Level: "info", Format: "console"}}

	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %q: %w", path, err)
		}
		dec := yaml.NewDecoder(strings.NewReader(string(raw)))
		dec.KnownFields(true)
		if err := dec.Decode(cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	}

	applyEnvOverrides(cfg)
	applyDefaults(cfg)

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func applyEnvOverrides(c *Config) {
	envStr := func(env string, dst *string) {
		if v := os.Getenv(env); v != "" {
			*dst = v
		}
	}
	envInt := func(env string, dst *int) {
		if v := os.Getenv(env); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	envStr("HYPERFOCUS_SERVICE_HTTP_ADDR", &c.Service.HTTPAddr)
	envStr("HYPERFOCUS_DB_HOST", &c.DB.Host)
	envInt("HYPERFOCUS_DB_PORT", &c.DB.Port)
	envStr("HYPERFOCUS_DB_USER", &c.DB.User)
	envStr("HYPERFOCUS_DB_PASSWORD", &c.DB.Password)
	envStr("HYPERFOCUS_DB_DATABASE", &c.DB.Database)
	envStr("HYPERFOCUS_DB_SSLMODE", &c.DB.SSLMode)
	envStr("HYPERFOCUS_TWITCH_CLIENT_ID", &c.Twitch.ClientID)
	envStr("HYPERFOCUS_TWITCH_CLIENT_SECRET", &c.Twitch.ClientSecret)
	envStr("HYPERFOCUS_TWITCHBOT_REFRESH_TOKEN", &c.TwitchBot.RefreshToken)
	envStr("HYPERFOCUS_STEAM_API_KEY", &c.Steam.APIKey)
	envInt("HYPERFOCUS_STEAM_RETRIES", &c.Steam.Retries)
	envInt("HYPERFOCUS_NOTIFY_WORKERS", &c.Notify.Workers)
	envStr("HYPERFOCUS_OCR_API_URL", &c.OCR.APIURL)
	envStr("HYPERFOCUS_STORAGE_DATA_DIR", &c.Storage.DataDir)
}

func set[T comparable](dst *T, def T) {
	var zero T
	if *dst == zero {
		*dst = def
	}
}

// applyDefaults fills any unset value. It only writes fields that are still at
// their zero value, so explicit YAML values are always preserved.
func applyDefaults(c *Config) {
	set(&c.Service.Name, "hyperfocus")
	set(&c.Service.HTTPAddr, ":8080")

	set(&c.DB.Host, "localhost")
	set(&c.DB.Port, 5432)
	set(&c.DB.User, "postgres")
	set(&c.DB.Database, "dbd")
	set(&c.DB.SSLMode, "prefer")
	set(&c.DB.MaxConns, int32(10))

	set(&c.Twitch.GameID, "491487") // Dead by Daylight

	if c.Poll.PageDelay == 0 {
		c.Poll.PageDelay = Duration(time.Second)
	}
	set(&c.Poll.PreviewWorkers, 16)
	if c.Poll.PreviewTimeout == 0 {
		c.Poll.PreviewTimeout = Duration(10 * time.Second)
	}
	set(&c.Poll.FetchMaxAttempts, 3)
	if c.Poll.FetchDelay == 0 {
		c.Poll.FetchDelay = Duration(2 * time.Second)
	}
	if c.Poll.PageSize <= 0 {
		c.Poll.PageSize = 100
	}

	if c.Prune.Interval == 0 {
		c.Prune.Interval = Duration(time.Hour)
	}
	set(&c.Prune.Hours, 72)

	set(&c.Storage.DataDir, "./data")

	if c.OCR.Enabled == nil {
		t := true
		c.OCR.Enabled = &t
	}
	set(&c.OCR.APIURL, "http://localhost:8081")
	if c.OCR.Workers <= 0 {
		c.OCR.Workers = 2
	}
	if c.OCR.Timeout == 0 {
		c.OCR.Timeout = Duration(15 * time.Second)
	}

	set(&c.Log.Level, "info")
	set(&c.Log.Format, "console")

	if c.Notify.Enabled == nil {
		f := false
		c.Notify.Enabled = &f
	}
	if c.Notify.MinScore == 0 {
		c.Notify.MinScore = 0.60
	}
	if c.Notify.Cooldown == 0 {
		c.Notify.Cooldown = Duration(30 * time.Minute)
	}
	set(&c.Notify.Workers, 2)

	if c.Steam.RefreshEvery == 0 {
		c.Steam.RefreshEvery = Duration(30 * time.Minute)
	}
	set(&c.Steam.Retries, 1)
}

// DSN builds a PostgreSQL connection string.
func (d DB) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode,
	)
}

// Duration is a time.Duration that unmarshals from YAML ("5m", or bare seconds).
type Duration time.Duration

// UnmarshalYAML implements yaml.v3.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	return d.set(s)
}

func (d *Duration) set(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		*d = 0
		return nil
	}
	// A bare integer is interpreted as seconds.
	if strings.TrimFunc(s, func(r rune) bool { return r >= '0' && r <= '9' }) == "" {
		var secs int
		if _, err := fmt.Sscanf(s, "%d", &secs); err != nil {
			return err
		}
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

// Std returns the underlying time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }
