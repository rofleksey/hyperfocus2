// Package config loads application configuration from a YAML file and validates
// it. Defaults are applied centrally for any unset values.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration object.
type Config struct {
	Service Service `yaml:"service"`
	DB      DB      `yaml:"db"`
	Twitch  Twitch  `yaml:"twitch"`
	Poll    Poll    `yaml:"poll"`
	Vod     Vod     `yaml:"vod"`
	Prune   Prune   `yaml:"prune"`
	Storage Storage `yaml:"storage"`
	OCR     OCR     `yaml:"ocr"`
	Log     Log     `yaml:"log"`
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
	PageSize           int      `yaml:"page_size"`
	PageDelay          Duration `yaml:"page_delay"`
	PreviewWidth       int      `yaml:"preview_width"`
	PreviewHeight      int      `yaml:"preview_height"`
	ThumbPreviewWidth  int      `yaml:"thumb_preview_width"`
	ThumbPreviewHeight int      `yaml:"thumb_preview_height"`
	PreviewWorkers     int      `yaml:"preview_workers"`
	PreviewTimeout     Duration `yaml:"preview_timeout"`
	FetchMaxAttempts   int      `yaml:"fetch_max_attempts"`
	FetchDelay         Duration `yaml:"fetch_delay"`
}

type Vod struct {
	ResolveInterval Duration `yaml:"resolve_interval"`
}

type Prune struct {
	Interval Duration `yaml:"interval"`
	Days     int      `yaml:"days"`
}

type Storage struct {
	DataDir string `yaml:"data_dir"`
}

// OCR configures the survivor-name OCR subprocess invoked once per poll cycle.
// The OCR pipeline is embedded into the Go binary and extracted at startup into
// a temporary directory — no external paths needed. python_bin is optional:
// leave empty (defaults to `python3` on PATH) for Docker; set it to a venv
// path for local dev. Enabled is a pointer so an explicit `enabled: false` in
// YAML is honored while an omitted field still defaults to enabled.
type OCR struct {
	Enabled       *bool  `yaml:"enabled"`
	PythonBin     string `yaml:"python_bin"`
	Workers       int    `yaml:"workers"`
	PythonWorkers int    `yaml:"python_workers"`
}

// IsEnabled reports whether OCR is enabled, defaulting to true when the field is
// unset.
func (o OCR) IsEnabled() bool { return o.Enabled == nil || *o.Enabled }

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
		if err := yaml.Unmarshal(raw, cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	}

	applyDefaults(cfg)

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// applyDefaults fills any unset value. It only writes fields that are still at
// their zero value, so explicit YAML values are always preserved.
func applyDefaults(c *Config) {
	setStr := func(dst *string, v string) {
		if strings.TrimSpace(*dst) == "" {
			*dst = v
		}
	}
	setInt := func(dst *int, v int) {
		if *dst == 0 {
			*dst = v
		}
	}
	setInt32 := func(dst *int32, v int32) {
		if *dst == 0 {
			*dst = v
		}
	}

	setStr(&c.Service.Name, "hyperfocus")
	setStr(&c.Service.HTTPAddr, ":8080")

	setStr(&c.DB.Host, "localhost")
	setInt(&c.DB.Port, 5432)
	setStr(&c.DB.User, "postgres")
	setStr(&c.DB.Database, "dbd")
	setStr(&c.DB.SSLMode, "disable")
	setInt32(&c.DB.MaxConns, 10)

	setStr(&c.Twitch.GameID, "491487") // Dead by Daylight

	if c.Poll.PageDelay == 0 {
		c.Poll.PageDelay = Duration(time.Second)
	}
	setInt(&c.Poll.PreviewWidth, 1280)
	setInt(&c.Poll.PreviewHeight, 720)
	setInt(&c.Poll.ThumbPreviewWidth, 480)
	setInt(&c.Poll.ThumbPreviewHeight, 270)
	setInt(&c.Poll.PreviewWorkers, 16)
	if c.Poll.PreviewTimeout == 0 {
		c.Poll.PreviewTimeout = Duration(10 * time.Second)
	}
	setInt(&c.Poll.FetchMaxAttempts, 3)
	if c.Poll.FetchDelay == 0 {
		c.Poll.FetchDelay = Duration(2 * time.Second)
	}

	if c.Vod.ResolveInterval == 0 {
		c.Vod.ResolveInterval = Duration(2 * time.Minute)
	}

	if c.Prune.Interval == 0 {
		c.Prune.Interval = Duration(time.Hour)
	}
	if c.Prune.Days <= 0 {
		c.Prune.Days = 3
	}

	setStr(&c.Storage.DataDir, "./data")

	if c.OCR.Enabled == nil {
		t := true
		c.OCR.Enabled = &t
	}
	setInt(&c.OCR.Workers, 1)
	setInt(&c.OCR.PythonWorkers, 2)

	setStr(&c.Log.Level, "info")
	setStr(&c.Log.Format, "console")
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
