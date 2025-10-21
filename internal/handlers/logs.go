package handlers

import (
	"archive/zip"
	"encoding/json"
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

// =============================
//   Список логов
// =============================

// GET /api/v2/logs — список логов приложения (только .log)
func ListAllLogs(w http.ResponseWriter, r *http.Request) {
	files, err := utils.DiscoverLogFiles(false)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "scan failed", nil)
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Modified.After(files[j].Modified)
	})

	type item struct {
		Name     string `json:"name"`
		Dir      string `json:"dir"`
		Path     string `json:"path"`
		Root     string `json:"root"`
		Size     int64  `json:"size"`
		Human    string `json:"human"`
		Modified string `json:"modified"`
	}

	var out []item
	for _, f := range files {
		out = append(out, item{
			Name:     f.Name,
			Dir:      f.Dir,
			Path:     f.Path,
			Root:     f.RootID,
			Size:     f.Size,
			Human:    utils.HumanSize(f.Size),
			Modified: utils.FormatTS(f.Modified),
		})
	}

	sendJSON(w, http.StatusOK, "OK", out)
}

// =============================
//   Архив всех логов
// =============================

// GET /api/v2/logs/download-all — объединяет все логи в ZIP
func DownloadAllLogs(w http.ResponseWriter, r *http.Request) {
	files, err := utils.DiscoverLogFiles(true)
	if err != nil || len(files) == 0 {
		sendJSON(w, http.StatusNotFound, "no logs found", nil)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		`attachment; filename="logs_all_`+time.Now().UTC().Format("20060102T150405")+`.zip"`)

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

// =============================
//   Просмотр хвоста лога
// =============================

// GET /api/v2/logs/tail?name=api.log&lines=200&format=json|raw[&root=local|sd|media-0]
func TailUnified(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := strings.TrimSpace(q.Get("name"))
	lines := parseIntDefault(q.Get("lines"), 200)
	format := strings.ToLower(q.Get("format"))
	rootHint := strings.TrimSpace(q.Get("root"))

	if name == "" {
		sendJSON(w, http.StatusBadRequest, "missing name", nil)
		return
	}

	li, err := utils.ResolveOneByName(name, rootHint)
	if err != nil {
		if strings.Contains(err.Error(), "ambiguous") {
			cands, _ := utils.FindLogsByName(name)
			type choice struct {
				Root string `json:"root"`
				Dir  string `json:"dir"`
				Path string `json:"path"`
			}
			var opts []choice
			for _, c := range cands {
				opts = append(opts, choice{Root: c.RootID, Dir: c.Dir, Path: c.Path})
			}
			sendJSON(w, http.StatusConflict, "multiple matches; specify root", opts)
			return
		}
		sendJSON(w, http.StatusNotFound, err.Error(), nil)
		return
	}

	f, err := utils.OpenSafe(li.Path)
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

// =============================
//   Скачать выбранные логи
// =============================

// POST /api/v2/logs/download
// Body: {"files":[{"name":"api.log","root":"local"},{"name":"Modbus_BEMP.log"}]}
type DownloadRequestItem struct {
	Name string `json:"name"`
	Root string `json:"root,omitempty"`
}
type DownloadRequest struct {
	Files []DownloadRequestItem `json:"files"`
}

func DownloadSelectedLogs(w http.ResponseWriter, r *http.Request) {
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 {
		sendJSON(w, http.StatusBadRequest, "invalid body; expect {files:[...]}", nil)
		return
	}

	type resolved struct {
		Path string
		Dir  string
		Name string
	}
	var toZip []resolved

	for _, item := range req.Files {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		li, err := utils.ResolveOneByName(item.Name, strings.TrimSpace(item.Root))
		if err != nil {
			if strings.Contains(err.Error(), "ambiguous") {
				cands, _ := utils.FindLogsByName(item.Name)
				type choice struct {
					Name string `json:"name"`
					Root string `json:"root"`
					Dir  string `json:"dir"`
					Path string `json:"path"`
				}
				var opts []choice
				for _, c := range cands {
					opts = append(opts, choice{
						Name: c.Name, Root: c.RootID, Dir: c.Dir, Path: c.Path,
					})
				}
				sendJSON(w, http.StatusConflict, "multiple matches; specify root for "+item.Name, opts)
				return
			}
			sendJSON(w, http.StatusNotFound, "not found: "+item.Name, nil)
			return
		}
		toZip = append(toZip, resolved{Path: li.Path, Dir: li.Dir, Name: li.Name})
	}

	if len(toZip) == 0 {
		sendJSON(w, http.StatusNotFound, "no files to archive", nil)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		`attachment; filename="logs_selected_`+time.Now().UTC().Format("20060102T150405")+`.zip"`)

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		zw := zip.NewWriter(pw)
		defer zw.Close()
		for _, f := range toZip {
			if err := addFileToZip(zw, f.Path, f.Dir); err != nil {
				log.Warn().Err(err).Str("file", f.Path).Msg("zip add failed")
			}
		}
	}()

	io.Copy(w, pr)
}

// =============================
//   Вспомогательные функции
// =============================

func addFileToZip(zw *zip.Writer, fullPath, baseDir string) error {
	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
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
