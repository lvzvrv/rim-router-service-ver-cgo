package utils

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ============================
//   Конфигурация фильтров
// ============================

var (
	extLogRe = regexp.MustCompile(`(?i)\.log$`) // только .log
)

// LogInfo описывает найденный лог-файл.
type LogInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	RootID   string    `json:"root_id"` // идентификатор корневой директории (local/sd)
}

// Root описывает корень, где могут лежать логи.
type Root struct {
	ID   string // "local", "sd"
	Path string // абсолютный путь к папке tir_logs
}

// ============================
//   Хуки для тестов
// ============================

// Позволяют тестам подменять логику поиска и открытия файлов
var (
	DiscoverLogFilesFunc = discoverLogFiles
	FindLogsByNameFunc   = findLogsByName
	ResolveOneByNameFunc = resolveOneByName
	OpenSafeFunc         = openSafe
)

// Обёртки для совместимости с продакшн-кодом
func DiscoverLogFiles(includeTimestamped bool) ([]LogInfo, error) {
	return DiscoverLogFilesFunc(includeTimestamped)
}

func FindLogsByName(name string) ([]LogInfo, error) {
	return FindLogsByNameFunc(name)
}

func ResolveOneByName(name, rootHint string) (LogInfo, error) {
	return ResolveOneByNameFunc(name, rootHint)
}

func OpenSafe(path string) (*os.File, error) {
	return OpenSafeFunc(path)
}

// ============================
//   Поиск корневых директорий
// ============================

func ListRoots() []Root {
	var roots []Root

	// 1. Локальная директория рядом с бинарником
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		local := filepath.Join(exeDir, "tir_logs")
		_ = os.MkdirAll(local, 0o755)
		roots = append(roots, Root{ID: "local", Path: local})
	}

	// 2. Поиск tir_logs на SD-карте или разделе в /mnt
	filepath.WalkDir("/mnt", func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, "tir_logs") {
			roots = append(roots, Root{ID: "sd", Path: p})
			return filepath.SkipDir
		}
		return nil
	})

	sort.SliceStable(roots, func(i, j int) bool { return roots[i].Path < roots[j].Path })
	return roots
}

func AllowedRoots() []string {
	rs := ListRoots()
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Path)
	}
	return out
}

// ============================
//   Утилиты
// ============================

func LooksLikeLog(name string) bool {
	return extLogRe.MatchString(strings.ToLower(filepath.Base(name)))
}

func WithinAllowedRoots(p string, roots []string) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}
	for _, root := range roots {
		rabs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if rel, err := filepath.Rel(rabs, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

func strconvI(i int) string {
	digits := "0123456789"
	if i == 0 {
		return "0"
	}
	res := make([]byte, 0, 8)
	for i > 0 {
		res = append([]byte{digits[i%10]}, res...)
		i /= 10
	}
	return string(res)
}

// ============================
//   Основные сценарии (реальные реализации)
// ============================

func discoverLogFiles(includeTimestamped bool) ([]LogInfo, error) {
	roots := ListRoots()
	var out []LogInfo

	for _, root := range roots {
		filepath.WalkDir(root.Path, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !LooksLikeLog(d.Name()) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			abs, _ := filepath.Abs(p)
			out = append(out, LogInfo{
				Path:     abs,
				Name:     d.Name(),
				Dir:      filepath.Dir(abs),
				Size:     info.Size(),
				Modified: info.ModTime(),
				RootID:   root.ID,
			})
			return nil
		})
	}
	return out, nil
}

func findLogsByName(name string) ([]LogInfo, error) {
	if strings.TrimSpace(name) == "" || !LooksLikeLog(name) {
		return nil, errors.New("invalid log file name")
	}
	all, err := discoverLogFiles(true)
	if err != nil {
		return nil, err
	}
	var out []LogInfo
	for _, li := range all {
		if strings.EqualFold(li.Name, name) {
			out = append(out, li)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RootID == out[j].RootID {
			return out[i].Modified.After(out[j].Modified)
		}
		if out[i].RootID == "local" {
			return true
		}
		return false
	})
	return out, nil
}

func resolveOneByName(name, rootHint string) (LogInfo, error) {
	list, err := findLogsByName(name)
	if err != nil {
		return LogInfo{}, err
	}
	if len(list) == 0 {
		return LogInfo{}, errors.New("log not found")
	}
	if rootHint == "" {
		return LogInfo{}, errors.New("root is required for this operation")
	}
	for _, li := range list {
		if li.RootID == rootHint {
			return li, nil
		}
	}
	return LogInfo{}, errors.New("no match for given root")
}

func openSafe(path string) (*os.File, error) {
	if !WithinAllowedRoots(path, AllowedRoots()) {
		return nil, errors.New("path not allowed")
	}
	return os.Open(path)
}
