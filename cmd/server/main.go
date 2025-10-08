package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"your-app/internal/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	// Initialize router
	r := chi.NewRouter()

	// Basic middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(zerologMiddleware(logger)) // Наш кастомный логгер

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
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
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
