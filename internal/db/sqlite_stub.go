//go:build !cgo

package db

import (
	"database/sql"
	"errors"
)

// OpenSQLite сообщает, что драйвер требует CGO, если бинарник собирается
// без поддержки CGO.
func OpenSQLite(string) (*sql.DB, error) {
	return nil, errors.New("SQLite support requires CGO (build with CGO_ENABLED=1)")
}
