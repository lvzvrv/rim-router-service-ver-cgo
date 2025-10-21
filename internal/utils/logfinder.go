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
	RootID   string    `json:"root_id"` // идентификатор корневой директории (local/sd/media-x)
}

// Root описывает корень, где могут лежать логи.
type Root struct {
	ID   string // "local", "sd", "media-0", "media-1", ...
	Path string // абсолютный путь к папке logs
}

// ============================
//   Поиск корневых директорий
// ============================

// ListRoots — возвращает ограниченный список директорий, где могут храниться логи приложения.
// 1) ./logs рядом с исполняемым файлом (ID: "local")
// 2) /mnt/mmcblk0p1/logs — стандартный путь для SD-карты (ID: "sd")
// 3) /media/*/logs и /run/media/*/logs — автосмонтированные носители (ID: "media-<index>")
func ListRoots() []Root {
	var roots []Root

	// 1. Локальная директория рядом с бинарником
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		local := filepath.Join(exeDir, "logs")
		roots = append(roots, Root{ID: "local", Path: local})
	}

	// 2. Стандартный путь SD-карты для embedded-устройств
	if _, err := os.Stat("/mnt/mmcblk0p1"); err == nil {
		roots = append(roots, Root{ID: "sd", Path: "/mnt/mmcblk0p1/logs"})
	}

	// 3. Дополнительные варианты автоподключения SD/USB
	// ВНИМАНИЕ: мы не сканируем всю систему; мы только собираем явные кандидаты папок logs.
	patterns := []string{"/media/*/logs", "/run/media/*/logs"}
	mi := 0
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			// сортируем для стабильности ID
			sort.Strings(matches)
			for _, m := range matches {
				roots = append(roots, Root{ID: "media-" + strconvI(mi), Path: m})
				mi++
			}
		}
	}

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
	// минимальная реализация без импорта strconv для компактности
	// (если хочешь — можно заменить на strconv.Itoa)
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
		// WalkDir по конкретной папке logs; если её нет — просто пропустим
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
	// стабильный порядок: сначала local, затем sd, затем media-*
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].RootID, out[j].RootID
		if ri == rj {
			return out[i].Modified.After(out[j].Modified)
		}
		// приоритет по ID
		order := func(id string) int {
			switch {
			case id == "local":
				return 0
			case id == "sd":
				return 1
			case strings.HasPrefix(id, "media-"):
				return 2
			default:
				return 3
			}
		}
		return order(ri) < order(rj)
	})
	return out, nil
}

// ResolveOneByName — выбрать один файл по имени и optional rootHint ("local","sd","media-0"...).
// Если дубликатов несколько и hint не задан, вернёт ошибку.
func ResolveOneByName(name, rootHint string) (LogInfo, error) {
	list, err := FindLogsByName(name)
	if err != nil {
		return LogInfo{}, err
	}
	if len(list) == 0 {
		return LogInfo{}, errors.New("log not found")
	}
	if rootHint == "" {
		if len(list) > 1 {
			return LogInfo{}, errors.New("ambiguous name; multiple matches")
		}
		return list[0], nil
	}
	for _, li := range list {
		if li.RootID == rootHint {
			return li, nil
		}
	}
	return LogInfo{}, errors.New("no match for given root hint")
}

// OpenSafe — безопасное открытие файла из разрешённых директорий.
func OpenSafe(path string) (*os.File, error) {
	if !WithinAllowedRoots(path, AllowedRoots()) {
		return nil, errors.New("path not allowed")
	}
	return os.Open(path)
}
