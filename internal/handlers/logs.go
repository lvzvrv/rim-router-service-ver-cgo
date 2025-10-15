package handlers

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"rim-router-service-ver-cgo/internal/utils"

	"github.com/rs/zerolog/log"
)

type LogListItem struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Dir      string `json:"dir"`
	Size     int64  `json:"size"`
	Human    string `json:"human"`
	Modified string `json:"modified"`
}

// GET /api/v2/logs — список логов по системе (без timestamp-архивов)
func ListAllLogs(w http.ResponseWriter, r *http.Request) {
	files, err := utils.DiscoverLogFiles(false)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "scan failed", nil)
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Modified.After(files[j].Modified) })

	var out []LogListItem
	for _, f := range files {
		out = append(out, LogListItem{
			Path:     f.Path,
			Name:     f.Name,
			Dir:      f.Dir,
			Size:     f.Size,
			Human:    utils.HumanSize(f.Size),
			Modified: utils.FormatTS(f.Modified),
		})
	}
	sendJSON(w, http.StatusOK, "OK", out)
}

// GET /api/v2/logs/download-all — объединяет все логи в один ZIP
func DownloadAllLogs(w http.ResponseWriter, r *http.Request) {
	files, err := utils.DiscoverLogFiles(true)
	if err != nil || len(files) == 0 {
		sendJSON(w, http.StatusNotFound, "no logs found", nil)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="logs_`+time.Now().UTC().Format("20060102T150405")+`.zip"`)

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		zw := zip.NewWriter(pw)
		defer zw.Close()

		for _, f := range files {
			if err := addFileToZip(zw, f.Path, f.Dir); err != nil {
				log.Warn().Err(err).Str("file", f.Path).Msg("zip add failed")
			}
		}
	}()

	io.Copy(w, pr)
}

// GET /api/v2/logs/download?path=...
func DownloadOneLog(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Query().Get("path"))
	if p == "" {
		sendJSON(w, http.StatusBadRequest, "missing path", nil)
		return
	}
	if _, err := os.Stat(p); err != nil {
		sendJSON(w, http.StatusNotFound, "file not found", nil)
		return
	}
	if !utils.WithinAllowedRoots(p, utils.AllowedRoots()) {
		sendJSON(w, http.StatusForbidden, "path not allowed", nil)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(p)+`.zip"`)

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		zw := zip.NewWriter(pw)
		defer zw.Close()
		addFileToZip(zw, p, filepath.Dir(p))
	}()

	io.Copy(w, pr)
}

// GET /api/v2/logs/tail?path=...&lines=...&format=json|raw
func TailUnified(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := q.Get("path")
	lines := parseIntDefault(q.Get("lines"), 200)
	format := strings.ToLower(q.Get("format"))

	if p == "" {
		sendJSON(w, http.StatusBadRequest, "missing path", nil)
		return
	}
	if _, err := os.Stat(p); err != nil {
		sendJSON(w, http.StatusNotFound, "file not found", nil)
		return
	}

	f, err := utils.OpenSafe(p)
	if err != nil {
		sendJSON(w, http.StatusForbidden, "open blocked", nil)
		return
	}
	defer f.Close()

	ctx := r.Context()
	type result struct {
		lines []string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		defer close(ch)
		ls, err := utils.ReadTailLines(f, lines)
		ch <- result{lines: ls, err: err}
	}()

	select {
	case <-ctx.Done():
		return
	case res := <-ch:
		if res.err != nil {
			sendJSON(w, http.StatusInternalServerError, "tail failed", nil)
			return
		}

		if format == "raw" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, strings.Join(res.lines, "\n"))
			return
		}

		normalized := make([]any, 0, len(res.lines))
		for _, s := range res.lines {
			ss := strings.TrimSpace(s)
			if strings.HasPrefix(ss, "{") && json.Valid([]byte(ss)) {
				normalized = append(normalized, json.RawMessage(ss))
			} else {
				normalized = append(normalized, utils.ParseBracketLine(ss))
			}
		}
		sendJSON(w, http.StatusOK, "OK", normalized)
	}
}

// addFileToZip добавляет файл в архив.
func addFileToZip(zw *zip.Writer, fullPath, baseDir string) error {
	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return errors.New("directory not supported")
	}

	rc, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer rc.Close()

	rel, err := filepath.Rel(baseDir, fullPath)
	if err != nil {
		rel = filepath.Base(fullPath)
	}

	h := &zip.FileHeader{
		Name:     filepath.ToSlash(rel),
		Method:   zip.Deflate,
		Modified: fi.ModTime(),
	}
	h.SetMode(0o644)

	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rc)
	return err
}

func parseIntDefault(s string, d int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return d
	}
	return n
}
