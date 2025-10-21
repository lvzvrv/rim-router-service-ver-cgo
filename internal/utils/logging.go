package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	PreferredSDPath  = "/mnt/mmcblk0p1/logs" // путь SD-карты
	LocalLogPath     = "./logs"              // локальная директория
	LogFileName      = "api.log"
	MaxLogSizeBytes  = 5 * 1024 * 1024 // 5 MB
	MaxArchivedFiles = 5               // максимум старых логов
	MinFreeSpaceMB   = 6               // минимум свободного места
)

var logDir string

// =============================
//   Инициализация каталога логов
// =============================

func ChooseLogDir() string {
	sdRoot := "/mnt/mmcblk0p1"
	sdLogs := filepath.Join(sdRoot, "logs")

	// Проверяем, смонтирована ли SD-карта
	if fi, err := os.Stat(sdRoot); err == nil && fi.IsDir() {
		if err := ensureDir(sdLogs); err == nil {
			logDir = sdLogs
			return logDir
		}
	}

	// Фолбэк в локальную директорию
	_ = ensureDir(LocalLogPath)
	logDir = LocalLogPath
	return logDir
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	testFile := filepath.Join(dir, ".test")
	if err := os.WriteFile(testFile, []byte("ok"), 0o644); err != nil {
		return err
	}
	_ = os.Remove(testFile)
	return nil
}

func LogDir() string {
	if logDir == "" {
		return ChooseLogDir()
	}
	return logDir
}

func LogFilePath() string {
	return filepath.Join(LogDir(), LogFileName)
}

// =============================
//   Реализация RotatingWriter
// =============================

type RotatingWriter struct {
	file *os.File
}

func NewRotatingWriter() (*RotatingWriter, error) {
	dir := LogDir()
	path := filepath.Join(dir, LogFileName)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return &RotatingWriter{file: f}, nil
}

func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	stat, err := w.file.Stat()
	if err != nil {
		return 0, err
	}

	// Проверяем место на диске
	if err := checkDiskSpaceAndCleanup(); err != nil {
		log.Error().
			Str("module", "system").
			Msgf("Insufficient disk space: %v — skipping log write", err)
		// Не пишем, чтобы не добить диск
		return 0, nil
	}

	// Проверяем размер файла
	if stat.Size()+int64(len(p)) > MaxLogSizeBytes {
		if err := w.rotate(); err != nil {
			log.Error().
				Str("module", "system").
				Msgf("Log rotation failed: %v", err)
		}
	}

	return w.file.Write(p)
}

func (w *RotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}

	oldPath := filepath.Join(LogDir(), LogFileName)
	timestamp := time.Now().UTC().Format("20060102T150405")
	newName := fmt.Sprintf("api.%s.log", timestamp)
	newPath := filepath.Join(LogDir(), newName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename log: %w", err)
	}

	f, err := os.OpenFile(oldPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("reopen log: %w", err)
	}
	w.file = f

	log.Info().
		Str("module", "system").
		Str("archived_log", newName).
		Msg("Log rotated successfully")

	cleanupOldLogs()
	return nil
}

func (w *RotatingWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// =============================
//   Очистка старых логов
// =============================

// cleanupOldLogs — удаляет только архивы вида api.YYYYMMDDTHHMMSS.log.
func cleanupOldLogs() {
	dir := LogDir()
	entries, _ := os.ReadDir(dir)

	var logs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == LogFileName {
			continue // не трогаем текущий api.log
		}
		// удаляем только наши архивы вида api.XXXXXX.log
		if filepath.Ext(name) == ".log" && len(name) > 8 && name[:4] == "api." {
			logs = append(logs, e)
		}
	}

	if len(logs) <= MaxArchivedFiles {
		return
	}

	sort.Slice(logs, func(i, j int) bool {
		ai, _ := logs[i].Info()
		aj, _ := logs[j].Info()
		return ai.ModTime().Before(aj.ModTime())
	})

	for i := 0; i < len(logs)-MaxArchivedFiles; i++ {
		os.Remove(filepath.Join(dir, logs[i].Name()))
	}
}

// =============================
//   Проверка свободного места
// =============================

func checkDiskSpaceAndCleanup() error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(LogDir(), &stat); err != nil {
		return err
	}
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	freeMB := float64(freeBytes) / (1024 * 1024)

	if freeMB < MinFreeSpaceMB {
		log.Warn().
			Str("module", "system").
			Float64("free_mb", freeMB).
			Msg("Low disk space detected: attempting cleanup")

		// Пробуем удалить старые архивы
		cleanupOldLogs()

		// Проверим ещё раз после очистки
		if err := syscall.Statfs(LogDir(), &stat); err != nil {
			return err
		}
		freeBytes = stat.Bavail * uint64(stat.Bsize)
		freeMB = float64(freeBytes) / (1024 * 1024)

		if freeMB < MinFreeSpaceMB {
			// Всё ещё мало — сообщаем об ошибке и блокируем запись
			return fmt.Errorf("low disk space (%.2f MB free, minimum %.2f MB required)",
				freeMB, MinFreeSpaceMB)
		}
	}
	return nil
}

// =============================
//   Вспомогательные функции
// =============================

func FormatTS(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func HumanSize(n int64) string {
	const kb = 1024
	const mb = 1024 * 1024
	switch {
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
