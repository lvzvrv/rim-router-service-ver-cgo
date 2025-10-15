package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	database "rim-router-service-ver-cgo/internal/db"
	"rim-router-service-ver-cgo/internal/handlers"
	myMiddleware "rim-router-service-ver-cgo/internal/middleware"
	"rim-router-service-ver-cgo/internal/models"
	"rim-router-service-ver-cgo/internal/utils"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configPath = flag.String("config", "", "path to config file")
	logLevel   = flag.String("log-level", "info", "log level")
)

func main() {
	flag.Parse()

	logger := setupLogger()
	defer logger.Info().Msg("Server shutdown")

	if err := loadConfig(*configPath, logger); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	dbConn, err := database.OpenSQLite("./data.db")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to open database")
	}
	defer dbConn.Close()

	if err := runMigrations(dbConn); err != nil {
		logger.Fatal().Err(err).Msg("Migrations failed")
	}

	userRepo := models.NewUserRepository(dbConn)
	tokenRepo := models.NewTokenRepository(dbConn)

	database.SeedAdmin(userRepo)

	authHandler := handlers.NewAuthHandler(userRepo, tokenRepo)

	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(zerologMiddleware(logger))

	r.Get("/health", handlers.HealthHandler)
	r.Post("/api/v1/register", authHandler.Register)
	r.Post("/api/v1/login", authHandler.Login)
	r.Post("/api/v1/refresh", authHandler.Refresh)
	r.Post("/api/v1/logout", authHandler.Logout)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(myMiddleware.AuthMiddleware)
		r.Get("/softwareVer", handlers.GetSoftwareVer)
	})

	// New log API (admin only)
	r.Route("/api/v2", func(r chi.Router) {
		r.Use(myMiddleware.AuthMiddleware)
		r.With(myMiddleware.RoleMiddleware(1)).Group(func(r chi.Router) {
			r.Get("/log", handlers.GetLogTail)
			r.Get("/loglist", handlers.GetLogList)
			r.Get("/log/download", handlers.DownloadLog)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Info().Msgf("Server listening on :%s", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatal().Err(err).Msg("Server failed")
	}
}

func setupLogger() zerolog.Logger {
	dir := utils.ChooseLogDir()

	writer, err := utils.NewRotatingWriter()
	if err != nil {
		fmt.Println("failed to init rotating writer:", err)
		os.Exit(1)
	}

	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339

	logger := zerolog.New(writer).With().Timestamp().Logger().Level(level)
	log.Logger = logger

	logger.Info().
		Str("module", "system").
		Str("log_dir", dir).
		Str("file", utils.LogFilePath()).
		Msg("Logging initialized")

	return logger
}

func zerologMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info().
				Str("method", r.Method).
				Str("url", r.URL.Path).
				Int("status", ww.Status()).
				Int("size", ww.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("request")
		})
	}
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_refresh_user_id ON refresh_tokens(user_id);
		CREATE INDEX IF NOT EXISTS idx_refresh_token ON refresh_tokens(token);
	`)
	return err
}

func loadConfig(path string, logger zerolog.Logger) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return fmt.Errorf("stat config file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("config path %s is a directory", trimmed)
	}
	file, err := os.Open(trimmed)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()
	logger.Info().Str("config", trimmed).Msg("Using configuration file")
	return nil
}
