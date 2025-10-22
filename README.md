#  ⚙️ Документация проекта 
## 📁 Структура проекта
```
rim-router-service-ver-cgo
├── build/                           # Собранные артефакты и база данных во время работы
│   ├── data.db                      # Основная база данных SQLite
│   ├── data.db-shm / data.db-wal    # Вспомогательные файлы SQLite (журнал транзакций)
│   ├── router-service               # Скомпилированный бинарный файл Go-сервера
│   └── tir_logs/                    # Папка с логами
│       └── api.log                  # Лог работы API

├── cmd/
│   └── server/
│       └── main.go                  # Точка входа в приложение (инициализация сервера и маршрутов)

├── internal/                        # Основная логика приложения
│   ├── config/
│   │   └── jwt.go                   # Настройки и функции генерации JWT-токенов
│   │
│   ├── db/
│   │   ├── sqlite_cgo.go            # Подключение и инициализация SQLite с поддержкой CGO
│   │   ├── sqlite_stub.go           # Альтернативная реализация без CGO (для сборки без зависимостей)
│   │   └── seed_admin.go            # Скрипт для автосоздания администратора при старте
│   │
│   ├── handlers/                    # HTTP-обработчики (эндпоинты REST API)
│   │   ├── auth.go                  # Регистрация, логин, refresh токенов
│   │   ├── logs.go                  # Работа с логами (просмотр, архивирование, скачивание)
│   │   ├── admin.go                 # Админские функции: пользователи, роли, управление
│   │   └── app.go                   # Общие обработчики (например, /health, версия ПО)
│   │
│   ├── middleware/                  # Промежуточные обработчики (middlewares)
│   │   └── auth.go                  # Проверка JWT, авторизация по ролям
│   │
│   ├── models/                      # Модели данных и работа с репозиториями
│   │   ├── user.go                  # Структура пользователя и методы работы с ним
│   │   └── token_repository.go      # Управление токенами и их хранением
│   │
│   └── utils/                       # Вспомогательные утилиты
│       ├── jwt.go                   # Общие функции по работе с JWT
│       ├── logging.go               # Настройка и форматирование логирования
│       ├── logfinder.go             # Поиск лог-файлов в системе
│       └── logparser.go             # Чтение и парсинг содержимого логов
│
├── migrations/                      # SQL-миграции для БД
│   ├── 001_create_users.up.sql      # Создание таблицы пользователей
│   └── 001_create_users.down.sql    # Откат миграции (удаление таблицы пользователей)
│
├── scripts/
│   └── db_tool.go                   # Утилита для обслуживания/манипуляций с БД вручную
│
├── Makefile                         # Команды сборки, сжатия и очистки проекта
├── go.mod                           # Зависимости и имя модуля
└── go.sum                           # Контрольные суммы зависимостей
```
## ⚙️ Файл `main.go`

Файл **`main.go`** является точкой входа в приложение и отвечает за полный цикл инициализации сервиса:

1. **Запускает HTTP-сервер** на базе `chi`-роутера  
2. **Настраивает логирование** через `zerolog` и `utils.NewRotatingWriter()`  
3. **Подключается к базе данных** SQLite (через `internal/db`)  
4. **Выполняет миграции** для создания таблиц  
5. **Инициализирует репозитории** и хендлеры (`authHandler`, `adminHandler`)  
6. **Определяет маршруты и middleware**

---

### 🔹 Запуск логгера

```go
logger := setupLogger()
defer logger.Info().Msg("Server shutdown")

if err := loadConfig(*configPath, logger); err != nil {
	logger.Fatal().Err(err).Msg("Failed to load config")
}
```
Настраивает систему логирования zerolog с автоматической ротацией логов.
В случае ошибок при загрузке конфигурации приложение аварийно завершится.

### 🔹 Подключение к базе данных
```go
dbConn, err := database.OpenSQLite("./data.db")
if err != nil {
	logger.Fatal().Err(err).Msg("Failed to open database")
}
defer dbConn.Close()
```
Открывает соединение с базой данных SQLite.
Используется единое подключение (*sql.DB) для всех репозиториев.

### 🔹 Выполнение миграций и создание первого администратора
```go
if err := runMigrations(dbConn); err != nil {
	logger.Fatal().Err(err).Msg("Migrations failed")
}

database.SeedAdmin(userRepo)
```
runMigrations() создаёт таблицы users и refresh_tokens, если их нет.
SeedAdmin() проверяет наличие администратора и создаёт его при первом запуске.

### 🔹 Инициализация репозиториев
```go
userRepo := models.NewUserRepository(dbConn)
tokenRepo := models.NewTokenRepository(dbConn)
Репозитории — слой работы с базой данных:
```
userRepo управляет таблицей пользователей (users),

tokenRepo управляет таблицей refresh-токенов (refresh_tokens).

Репозитории инкапсулируют SQL-логику и предоставляют удобные методы для работы с данными.

### 🔹 Инициализация хендлеров
```go
authHandler := handlers.NewAuthHandler(userRepo, tokenRepo)
adminHandler := handlers.NewAdminHandler(userRepo)
```
authHandler — обрабатывает запросы /api/v1/login, /register, /refresh, /logout

adminHandler — обрабатывает запросы /admin/users и /admin/users/{id}/role

Хендлеры — это слой API-логики, который взаимодействует с репозиториями и отвечает клиенту в формате JSON.

### 🔹 Настройка роутера и middleware
```go
r := chi.NewRouter()

r.Use(chimiddleware.RequestID)                 // добавляет уникальный ID каждому запросу
r.Use(chimiddleware.RealIP)                    // определяет реальный IP клиента
r.Use(chimiddleware.Recoverer)                 // ловит паники, чтобы сервер не падал
r.Use(chimiddleware.Timeout(60 * time.Second)) // ограничивает время выполнения запроса
r.Use(zerologMiddleware(logger))               // логирует метод, путь, статус и время
```
Middleware выполняются до каждого хендлера и обеспечивают безопасность, стабильность и наблюдаемость:

RequestID — присваивает каждому запросу уникальный ID

RealIP — определяет реальный IP клиента, даже если сервер стоит за прокси

Recoverer — перехватывает паники и предотвращает падение приложения

Timeout — автоматически завершает слишком долгие запросы

zerologMiddleware — записывает в лог метод, путь, статус и время выполнения запроса

### 🔹 Определение маршрутов
```go
// --- Public endpoints ---
r.Get("/health", handlers.HealthHandler)
r.Post("/api/v1/register", authHandler.Register)
r.Post("/api/v1/login", authHandler.Login)
r.Post("/api/v1/refresh", authHandler.Refresh)
r.Post("/api/v1/logout", authHandler.Logout)

// --- Authenticated v1 ---
r.Route("/api/v1", func(r chi.Router) {
	r.Use(myMiddleware.AuthMiddleware)
	r.Get("/softwareVer", handlers.GetSoftwareVer)
})

// --- Admin-only v2 (logs, user management, etc.) ---
r.Route("/api/v2", func(r chi.Router) {
	r.Use(myMiddleware.AuthMiddleware)
	r.With(myMiddleware.RoleMiddleware(1)).Group(func(r chi.Router) {
		// --- System logs ---
		r.Get("/logs", handlers.ListAllLogs)
		r.Get("/logs/download-all", handlers.DownloadAllLogs)
		r.Get("/logs/download", handlers.DownloadSelectedLogs)
		r.Get("/logs/tail", handlers.TailUnified)

		// --- User management ---
		r.Get("/admin/users", adminHandler.ListUsers)
		r.Post("/admin/users/{id}/role", adminHandler.UpdateUserRole)
	})
})
```
Роутер организован по уровням доступа:

Публичные маршруты — /health, /login, /register

Авторизованные маршруты — /api/v1/... (требуют JWT)

Админские маршруты — /api/v2/... (требуют роль администратора)

### 🔹 Логирование запросов
Функция setupLogger() настраивает zerolog с автоматической ротацией файлов через utils.NewRotatingWriter().
Каждый запрос логируется в формате:

```
метод | путь | статус | размер ответа | длительность
```
Логи сохраняются в папке build/tir_logs/ и автоматически ротируются при достижении лимита размера файла.

### 🔹 Функция runMigrations()
Создаёт базовые таблицы, если они отсутствуют:

users — хранит пользователей, их роли и дату создания

## 🔐 Конфигурация JWT (`internal/config/jwt.go`)

Отвечает за параметры генерации и проверки JWT-токенов.

```go
type JWTConfig struct {
    Secret            string
    AccessExpiration  time.Duration
    RefreshExpiration time.Duration
}
```

refresh_tokens — хранит refresh-токены, срок действия и связь с пользователем

Также создаются индексы для ускорения выборок по username и token.
