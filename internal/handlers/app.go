package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	tirStatus   bool
	tirStatusMu sync.RWMutex
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func sendJSON(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	response := Response{
		Code:    code,
		Message: message,
		Data:    data,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		fallback := `{"code": 500, "message": "JSON encoding error"}`
		_, _ = w.Write([]byte(fallback))
		return
	}

	_, _ = w.Write(jsonBytes)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"ok":      true,
		"message": "server is running",
		"ts":      time.Now().UnixMilli(),
	}

	logger := log.With().Str("module", "system").Logger()
	logger.Debug().Msg("Health check")

	_ = json.NewEncoder(w).Encode(response)
}

func GetSoftwareVer(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("module", "system").
		Str("endpoint", "/api/v1/softwareVer").
		Logger()
	(&logger).Info().Msg("Software version requested")

	sendJSON(w, http.StatusOK, "Success", "1.99.999")
}

func StartTir(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("module", "system").
		Str("endpoint", "/api/v2/startTir").
		Logger()

	tirStatusMu.Lock()
	defer tirStatusMu.Unlock()

	if tirStatus {
		(&logger).Warn().Msg("TIR already started")
		sendJSON(w, http.StatusBadRequest, "ТИР уже запущен", nil)
		return
	}

	tirStatus = true
	(&logger).Info().Msg("TIR started successfully")
	sendJSON(w, http.StatusOK, "ТИР успешно запущен", nil)
}

func StopTir(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("module", "system").
		Str("endpoint", "/api/v2/stopTir").
		Logger()

	tirStatusMu.Lock()
	defer tirStatusMu.Unlock()

	if !tirStatus {
		(&logger).Warn().Msg("TIR not running")
		sendJSON(w, http.StatusBadRequest, "ТИР не запущен", nil)
		return
	}

	tirStatus = false
	(&logger).Info().Msg("TIR stopped successfully")
	sendJSON(w, http.StatusOK, "ТИР успешно остановлен", nil)
}

func RestartTir(w http.ResponseWriter, r *http.Request) {
	logger := log.With().
		Str("module", "system").
		Str("endpoint", "/api/v2/restartTir").
		Logger()
	(&logger).Info().Msg("TIR restarted")

	sendJSON(w, http.StatusOK, "ТИР успешно перезапущен", nil)
}
