package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	database "your-app/internal/db"
	"your-app/internal/handlers"
	myMiddleware "your-app/internal/middleware"
	"your-app/internal/models"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	configPath = flag.String("config", "", "path to config file")
	logLevel   = flag.String("log-level", "info", "log level")
)

func main() {
	flag.Parse()

	// Setup logger
	logger := setupLogger()
	defer logger.Info().Msg("Server shutdown")

	// Load config file if provided
	if err := loadConfig(*configPath, logger); err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	// Initialize database using the CGO-enabled SQLite driver helper
	dbConn, err := database.OpenSQLite("./data.db")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to open database")
	}
	defer dbConn.Close()

	// Run migrations
	if err := runMigrations(dbConn); err != nil {
		logger.Fatal().Err(err).Msg("Migrations failed")
	}

	// Initialize repositories
	userRepo := models.NewUserRepository(dbConn)
	tokenRepo := models.NewTokenRepository(dbConn)

	// Create admin user if not exists
	database.SeedAdmin(userRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, tokenRepo)

	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(zerologMiddleware(logger))

	// Public routes
	r.Get("/health", handlers.HealthHandler)
	r.Post("/api/v1/register", authHandler.Register)
	r.Post("/api/v1/login", authHandler.Login)
	r.Post("/api/v1/refresh", authHandler.Refresh)
	r.Post("/api/v1/logout", authHandler.Logout)

	// Protected routes (require authentication)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(myMiddleware.AuthMiddleware)
		r.Get("/softwareVer", handlers.GetSoftwareVer)
	})

	// Start server
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
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	logWriter := &lumberjack.Logger{
		Filename:   "logs/server.log",
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     28,   // days
		Compress:   true, // compressed logs
	}

	logger := zerolog.New(logWriter).With().Timestamp().Logger().Level(level)
	return logger
}

// Middleware для логирования запросов
func zerologMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Создаем ResponseWriter для отслеживания статуса
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			// Логируем информацию о запросе
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
