// Package ocr is the adapter that extracts DbD survivor usernames from captured
// preview images. The actual image processing runs in a vendored Python pipeline
// (scripts/ocr/fastocr.py — RapidOCR / PaddleOCR-ONNX, tuned for the 1280x720
// preview resolution). This package shells out to it once per poll cycle with a
// batch of preview paths and parses the NDJSON results.
//
// The Python invocation is a single batched call so the ~0.2s ONNX model load
// is amortized across every image of the cycle. Failures are non-fatal: the
// poll continues with empty survivor names and logs the error.
//
// Timing is logged extensively: batch start/done with image counts, workers,
// total duration and per-image average, plus a per-image debug line. This lets
// an operator answer "how long does OCR take per image" and "does the batch fit
// inside the poll interval" directly from the logs.
package ocr

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/samber/oops"

	"hyperfocus/internal/config"
)

// ocrRecord is one NDJSON line emitted by `fastocr.py --json`.
type ocrRecord struct {
	Image string   `json:"image"`
	Names []string `json:"names"`
}

// Service runs the OCR subprocess. It is safe for concurrent use only if the
// caller serializes ExtractSurvivors calls (the poll usecase does).
type Service struct {
	log    *slog.Logger
	cfg    config.OCR
	script string // absolute path to fastocr.py
}

// New builds a Service. It resolves the fastocr.py path and, when OCR is
// enabled, warns (without failing) if the interpreter or script are missing so
// the application still starts and the poll loop can degrade gracefully.
func New(cfg config.OCR, log *slog.Logger) (*Service, error) {
	script := filepath.Join(cfg.ScriptDir, "fastocr.py")
	s := &Service{log: log, cfg: cfg, script: script}
	if !cfg.IsEnabled() {
		log.Info("ocr: disabled by configuration")
		return s, nil
	}
	// If the configured script is absent, try the bundled container path
	// (/opt/ocr) so the Docker image works without extra config. Falls through
	// to the original warning if neither exists.
	if _, err := os.Stat(script); err != nil {
		bundled := "/opt/ocr/fastocr.py"
		if _, err2 := os.Stat(bundled); err2 == nil {
			script = bundled
			s.script = bundled
			log.Info("ocr: using bundled script location", slog.String("path", bundled))
		}
	}
	if _, err := os.Stat(s.script); err != nil {
		log.Warn("ocr: fastocr.py not found at configured path; OCR will fail each cycle",
			slog.String("path", s.script),
			slog.String("script_dir", cfg.ScriptDir))
	}
	if _, err := exec.LookPath(cfg.PythonBin); err != nil {
		log.Warn("ocr: python interpreter not found on PATH; OCR will fail each cycle",
			slog.String("python_bin", cfg.PythonBin))
	}
	log.Info("ocr: configured",
		slog.String("python", cfg.PythonBin),
		slog.String("script", s.script),
		slog.Int("workers", cfg.Workers),
		slog.Duration("timeout", cfg.Timeout.Std()))
	return s, nil
}

// ExtractSurvivors runs the OCR pipeline over the given preview file paths and
// returns a map of path -> survivor names. Paths with no detections are omitted
// from the map. A non-nil error means the batch could not complete; partial
// results are still returned when available.
func (s *Service) ExtractSurvivors(ctx context.Context, paths []string) (map[string][]string, error) {
	out := make(map[string][]string, len(paths))
	if !s.cfg.IsEnabled() || len(paths) == 0 {
		return out, nil
	}

	timeout := s.cfg.Timeout.Std()
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	workers := s.cfg.Workers
	if workers <= 0 {
		workers = 1
	}

	args := make([]string, 0, len(paths)+4)
	args = append(args, s.script, "--json", "--workers", fmt.Sprintf("%d", workers))
	args = append(args, paths...)

	cmd := exec.CommandContext(cctx, s.cfg.PythonBin, args...)
	cmd.Stderr = os.Stderr // surface Python tracebacks / ONNX warnings in the app log

	pr, err := cmd.StdoutPipe()
	if err != nil {
		return out, oops.Wrap(err)
	}

	started := time.Now()
	s.log.Info("ocr: batch start",
		slog.Int("images", len(paths)),
		slog.Int("workers", workers),
		slog.String("python", s.cfg.PythonBin),
		slog.String("script", s.script),
		slog.Duration("timeout", timeout))

	if err := cmd.Start(); err != nil {
		s.log.Warn("ocr: failed to start subprocess", slog.Any("error", err))
		return out, oops.Wrap(err)
	}

	// Parse stdout as it streams so memory stays flat regardless of batch size.
	r := bufio.NewReaderSize(pr, 64*1024)
	for {
		line, readErr := r.ReadBytes('\n')
		if len(line) > 0 {
			s.handleLine(line, out)
		}
		if readErr != nil {
			if readErr != io.EOF {
				s.log.Debug("ocr: stdout read ended", slog.Any("error", readErr))
			}
			break
		}
	}

	waitErr := cmd.Wait()
	elapsed := time.Since(started)
	perImage := time.Duration(0)
	if n := len(paths); n > 0 {
		perImage = elapsed / time.Duration(n)
	}

	matched := len(out)
	failed := len(paths) - matched
	s.log.Info("ocr: batch done",
		slog.Int("images", len(paths)),
		slog.Int("ok", matched),
		slog.Int("no_names", failed),
		slog.Duration("duration", elapsed),
		slog.Duration("per_image_avg", perImage))

	if waitErr != nil {
		s.log.Warn("ocr: subprocess reported an error",
			slog.Any("error", waitErr),
			slog.Duration("duration", elapsed),
			slog.Bool("timed_out", cctx.Err() == context.DeadlineExceeded))
		return out, oops.Wrap(waitErr)
	}
	return out, nil
}

// handleLine parses one NDJSON record and records any non-empty names. Malformed
// lines are logged at debug and skipped so a single bad record can't abort the
// whole batch.
func (s *Service) handleLine(line []byte, out map[string][]string) {
	var rec ocrRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		s.log.Debug("ocr: skipping unparseable line", slog.String("line", string(line)))
		return
	}
	names := make([]string, 0, len(rec.Names))
	for _, n := range rec.Names {
		if n2 := trim(n); n2 != "" {
			names = append(names, n2)
		}
	}
	if rec.Image == "" || len(names) == 0 {
		return
	}
	out[rec.Image] = names
	s.log.Debug("ocr: image",
		slog.String("image", rec.Image),
		slog.Int("names", len(names)),
		slog.String("sample", names[0]))
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
