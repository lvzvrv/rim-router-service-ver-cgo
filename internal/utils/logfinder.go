package utils

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	isoTsRe       = regexp.MustCompile(`\d{8}T\d{6}`) // 20250919T052657
	extLogRe      = regexp.MustCompile(`(?i)\.log$`)  // только .log
	specialNames  = map[string]struct{}{"syslog": {}, "messages": {}, "dmesg": {}}
	excludedPaths = []string{"/usr/", "/lib/", "/snap/", "/opt/", "/proc/", "/sys/", "/dev/"}
)

// LogInfo описывает найденный файл лога.
type LogInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// AllowedRoots — ограниченный набор корней, где реально могут быть логи.
func AllowedRoots() []string {
	roots := []string{}
	roots = append(roots, LogDir()) // приоритетная директория приложения

	// системные и пользовательские каталоги
	candidates := []string{
		"/var/log",
		"/tmp",
	}

	// лог-каталоги на SD / media
	mountRoots := []string{"/mnt", "/media"}
	for _, root := range mountRoots {
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, "/logs") {
				candidates = append(candidates, path)
			}
			return nil
		})
	}

	roots = append(roots, candidates...)
	return roots
}

// IsTimestampedName — имя содержит ISO-метку времени (архивы логов).
func IsTimestampedName(name string) bool {
	return isoTsRe.MatchString(name)
}

// LooksLikeLog — определяет, является ли файл логом (строгое правило).
func LooksLikeLog(name string) bool {
	lower := strings.ToLower(filepath.Base(name))
	if extLogRe.MatchString(lower) {
		return true
	}
	// специальные имена без расширений (syslog, messages, dmesg)
	if _, ok := specialNames[lower]; ok {
		return true
	}
	return false
}

// isExcludedPath — путь попадает под исключение.
func isExcludedPath(p string) bool {
	for _, ex := range excludedPaths {
		if strings.HasPrefix(p, ex) {
			return true
		}
	}
	return false
}

// WithinAllowedRoots — проверяет, принадлежит ли путь разрешённым корням.
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

// DiscoverLogFiles — собирает только настоящие лог-файлы (.log, syslog и т.п.)
func DiscoverLogFiles(includeTimestamped bool) ([]LogInfo, error) {
	roots := AllowedRoots()
	seen := map[string]struct{}{}
	var out []LogInfo

	for _, root := range roots {
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			abs, _ := filepath.Abs(p)
			if seen[abs] != struct{}{} {
				seen[abs] = struct{}{}
			}

			// фильтрация по имени и расширению
			name := d.Name()
			if !LooksLikeLog(name) {
				return nil
			}
			if !includeTimestamped && IsTimestampedName(name) {
				return nil
			}

			// исключаем системные/вспомогательные каталоги
			if isExcludedPath(abs) {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			out = append(out, LogInfo{
				Path:     abs,
				Name:     name,
				Dir:      filepath.Dir(abs),
				Size:     info.Size(),
				Modified: info.ModTime(),
			})
			return nil
		})
	}
	return out, nil
}

// OpenSafe — безопасное открытие файла из разрешённых путей.
func OpenSafe(path string) (*os.File, error) {
	if !WithinAllowedRoots(path, AllowedRoots()) {
		return nil, errors.New("path not allowed")
	}
	return os.Open(path)
}
