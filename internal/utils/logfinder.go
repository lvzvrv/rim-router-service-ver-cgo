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
//   Поиск корневых директорий
// ============================

// ListRoots — возвращает ограниченный список директорий, где могут храниться логи приложения.
//  1. ./tir_logs рядом с исполняемым файлом (ID: "local")
//  2. /mnt/*/tir_logs — поиск по всем монтированным разделам SD/USB (ID: "sd")
//     (раньше был только /mnt/mmcblk0p1/logs, теперь ищем везде, но только tir_logs)
//     /media и /run/media исключены.
func ListRoots() []Root {
	var roots []Root

	// 1. Локальная директория рядом с бинарником
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		local := filepath.Join(exeDir, "tir_logs")
		_ = os.MkdirAll(local, 0o755) // создаём при необходимости
		roots = append(roots, Root{ID: "local", Path: local})
	}

	// 2. Поиск tir_logs на SD-карте или любом разделе в /mnt
	filepath.WalkDir("/mnt", func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		// Ищем только папки tir_logs
		if strings.HasSuffix(p, "tir_logs") {
			roots = append(roots, Root{ID: "sd", Path: p})
			return filepath.SkipDir // нашли — не спускаемся глубже
		}
		return nil
	})

	// Сортируем, чтобы порядок всегда был стабильным
	sort.SliceStable(roots, func(i, j int) bool { return roots[i].Path < roots[j].Path })
	return roots
}

// AllowedRoots — совместимая обёртка, если где-то в коде она уже используется.
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

// WithinAllowedRoots — проверяет, что путь принадлежит разрешённым корням.
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

// strconvI — маленькая утилита, чтобы не тянуть strconv везде
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
//   Основные сценарии
// ============================

// DiscoverLogFiles — ищет только файлы с расширением .log в ограниченных директориях.
func DiscoverLogFiles(includeTimestamped bool) ([]LogInfo, error) {
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

// FindLogsByName — найти все совпадения по имени файла среди допустимых корней.
func FindLogsByName(name string) ([]LogInfo, error) {
	if strings.TrimSpace(name) == "" || !LooksLikeLog(name) {
		return nil, errors.New("invalid log file name")
	}
	all, err := DiscoverLogFiles(true)
	if err != nil {
		return nil, err
	}
	var out []LogInfo
	for _, li := range all {
		if strings.EqualFold(li.Name, name) {
			out = append(out, li)
		}
	}
	// стабильный порядок: сначала local, затем sd
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

// ResolveOneByName — root обязателен.
func ResolveOneByName(name, rootHint string) (LogInfo, error) {
	list, err := FindLogsByName(name)
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

// OpenSafe — безопасное открытие файла из разрешённых директорий.
func OpenSafe(path string) (*os.File, error) {
	if !WithinAllowedRoots(path, AllowedRoots()) {
		return nil, errors.New("path not allowed")
	}
	return os.Open(path)
}
