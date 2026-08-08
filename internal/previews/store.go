// Package previews is the filesystem adapter for captured live-preview images.
// It downloads a CDN thumbnail, writes it under data/ as <uuid>.jpg, and prunes
// old files by mtime. The data dir is the single source of truth for bytes.
//
// In addition to the full-resolution preview, every saved image gets a small
// (480x270) "fast" thumbnail written under data/thumbs/<same-name>.jpg. The
// frontend gallery grid serves the fast thumbnail to keep the main page light;
// the per-stream modal serves the full-resolution preview.
package previews

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"

	"github.com/samber/oops"
)

// ThumbMaxWidth / ThumbMaxHeight are the bounding box of the fast thumbnail.
// The aspect ratio (16:9) matches the 1280x720 source so there is no cropping.
const (
	ThumbMaxWidth  = 480
	ThumbMaxHeight = 270
	thumbSubdir    = "thumbs"
)

// Store saves preview images to a directory and prunes them by age.
type Store struct {
	dir    string
	thumb  string
	client *http.Client
}

// New creates a Store, ensuring the main directory and the thumbs subdir exist.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create preview dir %q: %w", dir, err)
	}
	thumb := filepath.Join(dir, thumbSubdir)
	if err := os.MkdirAll(thumb, 0o750); err != nil {
		return nil, fmt.Errorf("create thumb dir %q: %w", thumb, err)
	}
	return &Store{
		dir:    dir,
		thumb:  thumb,
		client: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Dir returns the on-disk directory holding full-resolution previews.
func (s *Store) Dir() string { return s.dir }

// ThumbsDir returns the on-disk directory holding fast thumbnails.
func (s *Store) ThumbsDir() string { return s.thumb }

// Path returns the absolute path for a stored full-resolution filename.
func (s *Store) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

// ThumbPath returns the absolute path for a stored thumbnail filename.
func (s *Store) ThumbPath(filename string) string {
	return filepath.Join(s.thumb, filename)
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

// MakeThumbnail reads the full-resolution preview srcName, downscales it to fit
// within ThumbMaxWidth x ThumbMaxHeight (Catmull-Rom) and writes a JPEG quality
// 80 thumbnail under thumbs/<srcName>. It returns the source filename on
// success. A failure (e.g. corrupt source) is returned as an error; the caller
// treats thumbnail generation as best-effort.
func (s *Store) MakeThumbnail(srcName string) (string, error) {
	src, err := os.Open(s.Path(srcName))
	if err != nil {
		return "", oops.Wrap(err)
	}
	defer func() { _ = src.Close() }()
	img, err := jpeg.Decode(src)
	if err != nil {
		return "", oops.Wrap(err)
	}

	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return "", errors.New("empty image")
	}
	w, h := scaledSize(b.Dx(), b.Dy(), ThumbMaxWidth, ThumbMaxHeight)
	if w >= b.Dx() && h >= b.Dy() {
		// Source is already smaller than the thumbnail box; copy as-is.
		w, h = b.Dx(), b.Dy()
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Rect, img, b, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return "", oops.Wrap(err)
	}
	if err := os.WriteFile(s.ThumbPath(srcName), buf.Bytes(), 0o640); err != nil {
		return "", oops.Wrap(err)
	}
	return srcName, nil
}

// scaledSize computes the largest (w,h) within (maxW,maxH) preserving aspect.
func scaledSize(w, h, maxW, maxH int) (int, int) {
	rw := float64(maxW) / float64(w)
	rh := float64(maxH) / float64(h)
	r := rw
	if rh < r {
		r = rh
	}
	if r > 1 {
		r = 1
	}
	nw := int(float64(w)*r + 0.5)
	nh := int(float64(h)*r + 0.5)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return nw, nh
}

// Sweep removes preview files (and their thumbnails) older than olderThan by
// mtime. Returns the count of removed files across both directories.
func (s *Store) Sweep(_ context.Context, olderThan time.Time) (int, error) {
	removed, err := sweepDir(s.dir, olderThan)
	if err != nil {
		return removed, err
	}
	n, err := sweepDir(s.thumb, olderThan)
	return removed + n, err
}

// sweepDir removes top-level files older than cutoff. Subdirectories (such as
// the thumbs dir under the main dir) are skipped; their contents are swept by a
// dedicated call with that subdirectory as the target.
func sweepDir(dir string, cutoff time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
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
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}
