// Package previews is the filesystem adapter for captured live-preview images.
// It downloads a CDN thumbnail, writes it under data/ as <uuid>.jpg, and prunes
// old files by mtime. The data dir is the single source of truth for bytes.
package previews

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Store saves preview images to a directory and prunes them by age.
type Store struct {
	dir    string
	client *http.Client
}

// New creates a Store, ensuring the directory exists.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create preview dir %q: %w", dir, err)
	}
	return &Store{
		dir:    dir,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Dir returns the on-disk directory holding previews.
func (s *Store) Dir() string { return s.dir }

// Path returns the absolute path for a stored filename.
func (s *Store) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

// FetchAndSave downloads the image at url and stores it as <uuid>.jpg, returning
// the filename. A 0-length or non-JPEG response yields an error (no file written).
func (s *Store) FetchAndSave(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", errors.New("empty url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "hyperfocus/1.0 (+preview-capture)")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("preview fetch: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // 4 MiB cap
	if err != nil {
		return "", err
	}
	if len(body) < 64 || !bytes.HasPrefix(body, []byte{0xFF, 0xD8}) {
		return "", fmt.Errorf("preview fetch: not a jpeg (%d bytes)", len(body))
	}

	filename := uuid.NewString() + ".jpg"
	if err := os.WriteFile(s.Path(filename), body, 0o640); err != nil {
		return "", err
	}
	return filename, nil
}

// Sweep removes preview files older than olderThan by mtime.
func (s *Store) Sweep(_ context.Context, olderThan time.Time) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(olderThan) {
			if err := os.Remove(s.Path(e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}
