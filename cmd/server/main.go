package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"time"

	"your-app/internal/handlers"
	myMiddleware "your-app/internal/middleware"
	"your-app/internal/models"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "modernc.org/sqlite" // драйвер без CGO
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

	// Initialize database - ИЗМЕНИЛИ "sqlite3" на "sqlite"
	db, err := sql.Open("sqlite", "./data.db") // ← ИЗМЕНЕНИЕ ЗДЕСЬ
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to open database")
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		logger.Fatal().Err(err).Msg("Migrations failed")
	}

	// Initialize repositories
	userRepo := models.NewUserRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo)

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

	// Protected routes (require authentication)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(myMiddleware.AuthMiddleware)
		r.Get("/softwareVer", handlers.GetSoftwareVer)
	})

	// Admin routes (require admin role)
	r.Route("/api/v2", func(r chi.Router) {
		r.Use(myMiddleware.AuthMiddleware)
		r.Use(myMiddleware.RoleMiddleware(1)) // Только админы (role >= 1)

		r.Post("/startTir", handlers.StartTir)
		r.Post("/stopTir", handlers.StopTir)
		r.Post("/restartTir", handlers.RestartTir)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info().Str("port", port).Msg("Starting server")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatal().Err(err).Msg("Server failed")
	}
}

func setupLogger() zerolog.Logger {
	// Создаем папку для логов если её нет
	os.MkdirAll("logs", 0755)

	// Настройка ротации логов
	lumberjackLogger := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    5, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	// Multi writer: файл и консоль
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	multi := zerolog.MultiLevelWriter(consoleWriter, lumberjackLogger)

	logger := zerolog.New(multi).With().Timestamp().Logger()

	// Устанавливаем уровень логирования
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		logger.Warn().Str("level", *logLevel).Msg("Invalid log level, using info")
		level = zerolog.InfoLevel
	}
	logger = logger.Level(level)

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
	// Простая миграция для демо - в продакшене используйте github.com/golang-migrate/migrate
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`)
	return err
}
