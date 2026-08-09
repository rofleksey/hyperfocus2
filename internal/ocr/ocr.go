// Package ocr is the adapter that extracts DbD survivor usernames from captured
// preview images. The actual image processing runs in a vendored Python pipeline
// (RapidOCR / PaddleOCR-ONNX) that is embedded into the Go binary and extracted
// to a temporary directory at startup — no external script path or Python
// interpreter configuration is needed beyond ensuring `python3` is on PATH.
//
// Extraction is single-batch per poll cycle so the model load (~0.2 s) is
// amortized; failures are non-fatal and the poll continues with empty survivor
// names.
//
// Extensive timing logs (batch start/done with image counts, per-image average,
// per-image debug) are written so operators can measure whether OCR fits inside
// the poll interval.
package ocr

import (
	"bufio"
	"context"
	"encoding/json"
	_ "embed"
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

//go:embed fastocr.py
var fastocrScript string

//go:embed extract.py
var extractScript string

// ocrRecord is one NDJSON line emitted by the pipeline.
type ocrRecord struct {
	Image string   `json:"image"`
	Names []string `json:"names"`
}

// Service shells out to the embedded Python OCR pipeline. Concurrent use must
// be serialized by the caller (the poll usecase does this naturally).
type Service struct {
	log  *slog.Logger
	cfg  config.OCR
	path string // temp directory holding the extracted scripts
}

// New builds a Service. It extracts the embedded Python scripts to a temporary
// directory and warns (without failing) when python3 is missing so the
// application starts cleanly and the poll loop can degrade.
func New(cfg config.OCR, log *slog.Logger) (*Service, error) {
	s := &Service{log: log, cfg: cfg}
	if !cfg.IsEnabled() {
		log.Info("ocr: disabled by configuration")
		return s, nil
	}

	tmp, err := os.MkdirTemp("", "hf-ocr")
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "fastocr.py"), []byte(fastocrScript), 0o500); err != nil {
		return nil, oops.Wrap(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "extract.py"), []byte(extractScript), 0o400); err != nil {
		return nil, oops.Wrap(err)
	}
	s.path = tmp

	pyBin := s.cfg.PythonBin
	if pyBin == "" {
		pyBin = "python3"
	}
	if _, err := exec.LookPath(pyBin); err != nil {
		log.Warn("ocr: python interpreter not found on PATH; OCR will fail each cycle",
			slog.String("python_bin", pyBin))
	}
	log.Info("ocr: configured",
		slog.String("python", pyBin),
		slog.String("script_dir", tmp),
		slog.Int("workers", cfg.Workers),
		slog.Int("python_workers", cfg.PythonWorkers))
	return s, nil
}

// ExtractSurvivors runs the embedded OCR pipeline over the given preview file
// paths. It returns a map of path → survivor names. Paths with no detections
// are omitted. A non-nil error means the batch did not complete; partial results
// are still returned when available.
func (s *Service) ExtractSurvivors(ctx context.Context, paths []string) (map[string][]string, error) {
	out := make(map[string][]string, len(paths))
	if !s.cfg.IsEnabled() || len(paths) == 0 {
		return out, nil
	}

	workers := s.cfg.PythonWorkers
	if workers <= 0 {
		workers = 1
	}

	script := filepath.Join(s.path, "fastocr.py")
	args := make([]string, 0, len(paths)+4)
	args = append(args, script, "--json", "--workers", fmt.Sprintf("%d", workers))
	args = append(args, paths...)

	pyBin := s.cfg.PythonBin
	if pyBin == "" {
		pyBin = "python3"
	}
	cmd := exec.CommandContext(ctx, pyBin, args...)
	cmd.Stderr = os.Stderr

	pr, err := cmd.StdoutPipe()
	if err != nil {
		return out, oops.Wrap(err)
	}

	started := time.Now()
	s.log.Debug("ocr: image start",
		slog.Int("images", len(paths)),
		slog.Int("workers", workers))

	if err := cmd.Start(); err != nil {
		s.log.Warn("ocr: failed to start subprocess", slog.Any("error", err))
		return out, oops.Wrap(err)
	}

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

	matched := len(out)
	s.log.Debug("ocr: image done",
		slog.Int("images", len(paths)),
		slog.Int("ok", matched),
		slog.Int("no_names", len(paths)-matched),
		slog.Duration("duration", elapsed))

	if waitErr != nil {
		s.log.Warn("ocr: subprocess reported an error",
			slog.Any("error", waitErr),
			slog.Duration("duration", elapsed))
		return out, oops.Wrap(waitErr)
	}
	return out, nil
}

// handleLine parses one NDJSON record and records any non-empty names.
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
