package handlers

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	tirStatus   bool
	tirStatusMu sync.RWMutex
)

// Базовая функция для отправки JSON ответов
func sendJSON(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if data != nil {
		response := fmt.Sprintf(`{"code": %d, "message": "%s", "data": "%s"}`, code, message, data)
		w.Write([]byte(response))
	} else {
		response := fmt.Sprintf(`{"code": %d, "message": "%s"}`, code, message)
		w.Write([]byte(response))
	}
}

func GetSoftwareVer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GET /api/v1/softwareVer")
	sendJSON(w, http.StatusOK, "Success", "1.99.999")
}

func StartTir(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /api/v2/startTir")

	tirStatusMu.Lock()
	defer tirStatusMu.Unlock()

	if tirStatus {
		sendJSON(w, http.StatusBadRequest, "ТИР уже запущен", nil)
		return
	}

	tirStatus = true
	sendJSON(w, http.StatusOK, "ТИР успешно запущен", nil)
}

func StopTir(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /api/v2/stopTir")

	tirStatusMu.Lock()
	defer tirStatusMu.Unlock()

	if !tirStatus {
		sendJSON(w, http.StatusBadRequest, "ТИР не запущен", nil)
		return
	}

	tirStatus = false
	sendJSON(w, http.StatusOK, "ТИР успешно остановлен", nil)
}

func RestartTir(w http.ResponseWriter, r *http.Request) {
	fmt.Println("POST /api/v2/restartTir")
	sendJSON(w, http.StatusOK, "ТИР успешно перезапущен", nil)
}
