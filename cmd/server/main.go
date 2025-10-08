package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"your-app/internal/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	configPath = flag.String("config", "", "path to config file")
)

func main() {
	flag.Parse()

	fmt.Println("Starting server...")

	// Initialize router
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger) // простой логгер для начала

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true, "message": "server is running"}`))
	})

	// API routes v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/softwareVer", handlers.GetSoftwareVer)
	})

	// API routes v2
	r.Route("/api/v2", func(r chi.Router) {
		r.Post("/startTir", handlers.StartTir)
		r.Post("/stopTir", handlers.StopTir)
		r.Post("/restartTir", handlers.RestartTir)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed:", err)
	}
}
