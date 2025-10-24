package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rim-router-service-ver-cgo/internal/utils"

	"github.com/stretchr/testify/assert"
)

// ================================
//  Вспомогательные функции
// ================================

func makeTempLogFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	assert.NoError(t, err)
	return path
}

func readZipEntries(t *testing.T, data []byte) []string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	assert.NoError(t, err)
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

// ================================
//  Тест ListAllLogs
// ================================

func TestListAllLogs_Success(t *testing.T) {
	oldDiscover := utils.DiscoverLogFilesFunc
	defer func() { utils.DiscoverLogFilesFunc = oldDiscover }()

	utils.DiscoverLogFilesFunc = func(includeArchives bool) ([]utils.LogInfo, error) {
		return []utils.LogInfo{
			{
				Name:     "api.log",
				Path:     "/tmp/api.log",
				Dir:      "/tmp",
				Size:     123,
				Modified: time.Now(),
				RootID:   "local",
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/logs", nil)
	w := httptest.NewRecorder()

	ListAllLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "OK", resp.Message)
	assert.NotNil(t, resp.Data)
}

// ================================
//  Тест DownloadSelectedLogs
// ================================

func TestDownloadSelectedLogs_Success(t *testing.T) {
	tmpDir := t.TempDir()
	mockFile := makeTempLogFile(t, tmpDir, "api.log", "hello log")

	oldResolve := utils.ResolveOneByNameFunc
	defer func() { utils.ResolveOneByNameFunc = oldResolve }()

	utils.ResolveOneByNameFunc = func(name, root string) (utils.LogInfo, error) {
		return utils.LogInfo{Name: name, Path: mockFile, RootID: root}, nil
	}

	body := `{"files":[{"name":"api.log","root":"local"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs/download", strings.NewReader(body))
	w := httptest.NewRecorder()

	DownloadSelectedLogs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "zip")

	data := w.Body.Bytes()
	files := readZipEntries(t, data)
	assert.Equal(t, []string{"local/api.log"}, files)
}

func TestDownloadSelectedLogs_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/logs/download", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	DownloadSelectedLogs(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ================================
//  Тест TailUnified
// ================================

func TestTailUnified_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	mockFile := makeTempLogFile(t, tmpDir, "api.log", "[2025-10-24 10:00:00,000] [INFO] System::Start: OK")

	oldResolve := utils.ResolveOneByNameFunc
	oldOpen := utils.OpenSafeFunc
	oldReadTail := utils.ReadTailLinesFunc
	defer func() {
		utils.ResolveOneByNameFunc = oldResolve
		utils.OpenSafeFunc = oldOpen
		utils.ReadTailLinesFunc = oldReadTail
	}()

	utils.ResolveOneByNameFunc = func(name, root string) (utils.LogInfo, error) {
		return utils.LogInfo{Name: name, Path: mockFile, RootID: root}, nil
	}

	utils.OpenSafeFunc = func(path string) (*os.File, error) {
		return os.Open(mockFile)
	}

	utils.ReadTailLinesFunc = func(f *os.File, lines int) ([]string, error) {
		return []string{"[2025-10-24 10:00:00,000] [INFO] System::Start: OK"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/logs/tail?name=api.log&lines=10&format=json&root=local", nil)
	w := httptest.NewRecorder()

	TailUnified(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"OK"`)
}

func TestTailUnified_RawFormat(t *testing.T) {
	tmpDir := t.TempDir()
	mockFile := makeTempLogFile(t, tmpDir, "api.log", "line1\nline2")

	oldResolve := utils.ResolveOneByNameFunc
	oldOpen := utils.OpenSafeFunc
	oldReadTail := utils.ReadTailLinesFunc
	defer func() {
		utils.ResolveOneByNameFunc = oldResolve
		utils.OpenSafeFunc = oldOpen
		utils.ReadTailLinesFunc = oldReadTail
	}()

	utils.ResolveOneByNameFunc = func(name, root string) (utils.LogInfo, error) {
		return utils.LogInfo{Name: name, Path: mockFile, RootID: root}, nil
	}
	utils.OpenSafeFunc = func(path string) (*os.File, error) {
		return os.Open(mockFile)
	}
	utils.ReadTailLinesFunc = func(f *os.File, lines int) ([]string, error) {
		return []string{"line1", "line2"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/logs/tail?name=api.log&lines=2&format=raw&root=local", nil)
	w := httptest.NewRecorder()

	TailUnified(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "line1")
	assert.Contains(t, w.Body.String(), "line2")
}

func TestTailUnified_MissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/logs/tail", nil)
	w := httptest.NewRecorder()

	TailUnified(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ================================
//  Тест addFileToZipWithRoot
// ================================

func TestAddFileToZipWithRoot(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := makeTempLogFile(t, tmpDir, "test.log", "hello")

	outFile := filepath.Join(tmpDir, "out.zip")
	out, err := os.Create(outFile)
	assert.NoError(t, err)
	defer out.Close()

	zw := zip.NewWriter(out)
	err = addFileToZipWithRoot(zw, filePath, "local")
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	data, err := os.ReadFile(outFile)
	assert.NoError(t, err)

	files := readZipEntries(t, data)
	assert.Equal(t, []string{"local/test.log"}, files)
}

// ================================
//  Тест parseIntDefault
// ================================

func TestParseIntDefault(t *testing.T) {
	assert.Equal(t, 10, parseIntDefault("10", 5))
	assert.Equal(t, 5, parseIntDefault("", 5))
	assert.Equal(t, 5, parseIntDefault("-1", 5))
	assert.Equal(t, 5, parseIntDefault("abc", 5))
}
