// Package ocr is the HTTP client for the external survivor-name OCR
// microservice (hyperfocus2-ocr). Each preview image is POSTed individually as
// raw JPEG bytes; the microservice keeps the RapidOCR / PaddleOCR-ONNX model
// resident so the (relatively expensive) ONNX model load never happens per
// request. Failures are non-fatal and the poll continues with empty survivor
// names.
package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/config"
)

// Service is the HTTP client for the OCR microservice.
type Service struct {
	log    *slog.Logger
	cfg    config.OCR
	client *http.Client
	base   string // microservice base URL, no trailing slash
}

// New builds a Service pointed at the OCR microservice configured by cfg.APIURL.
func New(cfg config.OCR, log *slog.Logger) (*Service, error) {
	s := &Service{
		log:    log,
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout.Std()},
		base:   strings.TrimRight(cfg.APIURL, "/"),
	}
	if !cfg.IsEnabled() {
		log.Info("ocr: disabled by configuration")
		return s, nil
	}
	log.Info("ocr: configured",
		slog.String("api_url", s.base),
		slog.Duration("timeout", cfg.Timeout.Std()),
		slog.Int("workers", cfg.Workers))
	return s, nil
}

// ocrResponse is the JSON body returned by the microservice.
type ocrResponse struct {
	Names []string `json:"names"`
}

// ExtractSurvivors sends a single preview image to the OCR microservice and
// returns the recognized survivor nicknames. A nil slice with no error means
// OCR is disabled or the image yielded no names.
func (s *Service) ExtractSurvivors(ctx context.Context, path string) ([]string, error) {
	if !s.cfg.IsEnabled() {
		return nil, nil
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return nil, oops.Wrap(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/ocr", bytes.NewReader(body))
	if err != nil {
		return nil, oops.Wrap(err)
	}
	req.Header.Set("Content-Type", "image/jpeg")

	started := time.Now()
	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Warn("ocr: request failed",
			slog.String("image", path),
			slog.Any("error", err))
		return nil, oops.Wrap(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		s.log.Warn("ocr: microservice returned non-200",
			slog.String("image", path),
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(rb)))
		return nil, oops.Errorf("ocr: status %d", resp.StatusCode)
	}

	var out ocrResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		s.log.Warn("ocr: failed to decode response",
			slog.String("image", path), slog.Any("error", err))
		return nil, oops.Wrap(err)
	}

	names := make([]string, 0, len(out.Names))
	for _, n := range out.Names {
		if n2 := strings.TrimSpace(n); n2 != "" {
			names = append(names, n2)
		}
	}
	s.log.Debug("ocr: image",
		slog.String("image", path),
		slog.Int("names", len(names)),
		slog.Duration("duration", time.Since(started)))
	return names, nil
}
