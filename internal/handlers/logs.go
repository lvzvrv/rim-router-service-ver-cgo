package handlers

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"rim-router-service-ver-cgo/internal/utils"
)

type LogFileInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Human    string `json:"human"`
	Modified string `json:"modified"`
}

func GetLogTail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lines := parseIntDefault(q.Get("lines"), 200)
	format := q.Get("format")

	path := utils.LogFilePath()
	f, err := os.Open(path)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to open log", nil)
		return
	}
	defer f.Close()

	linesData, raw, err := tailLines(f, lines)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to read log", nil)
		return
	}

	if format == "raw" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(raw))
	} else {
		sendJSON(w, http.StatusOK, "OK", linesData)
	}
}

func GetLogList(w http.ResponseWriter, r *http.Request) {
	dir := utils.LogDir()
	entries, _ := os.ReadDir(dir)

	var logs []LogFileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "api") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		logs = append(logs, LogFileInfo{
			Name:     name,
			Size:     info.Size(),
			Human:    utils.HumanSize(info.Size()),
			Modified: utils.FormatTS(info.ModTime()),
		})
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Modified > logs[j].Modified
	})

	sendJSON(w, http.StatusOK, "OK", logs)
}

func DownloadLog(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Query().Get("name"))
	if name == "" {
		sendJSON(w, http.StatusBadRequest, "Missing 'name' param", nil)
		return
	}
	full := filepath.Join(utils.LogDir(), name)
	if _, err := os.Stat(full); err != nil {
		sendJSON(w, http.StatusNotFound, "File not found", nil)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	http.ServeFile(w, r, full)
}

// tailLines — безопасное чтение последних N строк без загрузки всего файла
func tailLines(f *os.File, n int) ([]string, string, error) {
	const chunk = 4 * 1024
	fi, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	size := fi.Size()
	if size == 0 {
		return []string{}, "", nil
	}
	var buf []byte
	var read int64 = 0
	for read < size && len(strings.Split(string(buf), "\n")) <= n {
		step := int64(chunk)
		if read+step > size {
			step = size - read
		}
		read += step
		tmp := make([]byte, step)
		_, err := f.ReadAt(tmp, size-read)
		if err != nil && !errors.Is(err, io.EOF) {
			break
		}
		buf = append(tmp, buf...)
	}
	lines := strings.Split(string(buf), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	raw := strings.Join(lines, "\n")
	return lines, raw, nil
}

func parseIntDefault(s string, d int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return d
	}
	return n
}
