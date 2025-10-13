//go:build cgo

package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultBusyTimeout = 5000 // milliseconds
)

// OpenSQLite открывает подключение к SQLite базе данных используя CGO-драйвер.
// Путь нормализуется, а соединение настраивается для экономии ресурсов
// встроенного устройства: ограничивается одним соединением и включаются
// важные pragma для надежности.
func OpenSQLite(path string) (*sql.DB, error) {
	if path == "" {
		path = "./data.db"
	}

	dsn, err := buildDSN(path)
	if err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// На встраиваемом устройстве оставляем одно активное соединение и
	// освобождаем ресурсы быстрее.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	return database, nil
}

func buildDSN(path string) (string, error) {
	params := buildDSNParams()

	if path == ":memory:" {
		return fmt.Sprintf("%s?%s", path, params), nil
	}

	if strings.HasPrefix(path, "file:") {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		return fmt.Sprintf("%s%s%s", path, separator, params), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve sqlite path: %w", err)
	}

	return fmt.Sprintf("file:%s?%s", url.PathEscape(absPath), params), nil
}

func buildDSNParams() string {
	values := url.Values{}
	values.Set("_busy_timeout", fmt.Sprintf("%d", defaultBusyTimeout))
	values.Set("_foreign_keys", "ON")
	values.Set("_journal_mode", "WAL")
	values.Set("cache", "shared")
	values.Set("mode", "rwc")
	return values.Encode()
}
