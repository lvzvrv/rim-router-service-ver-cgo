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
# ⚙️ Файл `main.go`

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

### 🗂️Функция runMigrations()
Создаёт базовые таблицы, если они отсутствуют:

users — хранит пользователей, их роли и дату создания

# 🔐 Конфигурация JWT (`internal/config/jwt.go`)

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

# 🗄️ Подключение к базе данных (`internal/db/sqlite_cgo.go`)
Использует CGO-драйвер `github.com/mattn/go-sqlite3` для работы с SQLite.
| Элемент                           | Назначение                                                            |
| --------------------------------- | --------------------------------------------------------------------- |
| `//go:build cgo`                  | Компилируется только при включённом CGO (использует C-драйвер SQLite) |
| `OpenSQLite(path)`                | Открывает соединение с SQLite, настраивает пул и параметры            |
| `buildDSN()`                      | Формирует корректную строку подключения к базе                        |
| `buildDSNParams()`                | Добавляет параметры (timeout, WAL, foreign keys)                      |
| `SetMaxOpenConns(1)`              | Ограничивает количество соединений для экономии ресурсов              |

#  📘 handlers/admin.go — обработчики административных запросов
Структура, объединяющая зависимости для всех эндпоинтов:
```go
type AdminHandler struct {
	UserRepo *models.UserRepository
	logger   zerolog.Logger
}
```

###  🗂️Функция ```func NewAdminHandler(userRepo *models.UserRepository) *AdminHandler```

Конструктор обработчика. Создает новый экзмепляр AdminHandler, настраивая логирование.


###  🗂️Функция ```func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request)```

Запрос ``` GET http://localhost:8080/api/v2/admin/users ```
Возвращает список всех пользователей.
Порядок работы:
1) Логирует вызов эндпоинта.
2) Обращается к UserRepo.GetAllUsers().
3) При ошибке возвращает 500 с сообщением "Database error".
4) При успехе возвращает 200 OK и массив пользователей в JSON-виде:
```json
{
  "code": 200,
  "message": "OK",
  "data": [
    {"id":1,"username":"admin","role":1,"created_at":"..."},
    {"id":2,"username":"user1","role":0,"created_at":"..."}
  ]
}
```


###  🗂️Функция ```func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request)```

Запрос ``` POST http://localhost:8080/api/v2/admin/users/3/role```
Изменяет роль пользователя по его id.
Ожидаемое тело запроса:
```json
{ "role": 1 }
```
Логика работы:
1) Извлекает id из URL ```go (chi.URLParam(r, "id"))```, проверяет что это положительное число.
Ошибка → ```400 Invalid user ID```.

2) Парсит JSON-тело в структуру ```{ Role int }```.
Ошибка → ```400 Invalid JSON```.

3) Проверяет допустимость значения Role (0–2).
Ошибка → ```400 Invalid role value```.

4) Вызывает ```go UserRepo.UpdateUserRole(id, role)```.
Ошибка БД → ```500 Failed to update role```.

5) При успехе пишет в лог:
```json
{"level":"info","module":"admin","user_id":5,"new_role":1,"msg":"User role updated successfully"}
```

6) Возвращает 200 OK и сообщение "Role updated successfully".


#  📘 handlers/auth.go — обработчики аутентификации и авторизации
Реализует полный цикл авторизации пользователей: регистрация, вход, обновление и выход из системы.
Хранит refresh_token в базе данных и в HttpOnly cookie для безопасного обновления access-токена.

### Структура обработчика:
```go
type AuthHandler struct {
	UserRepo  *models.UserRepository
	TokenRepo *models.TokenRepository
}

func NewAuthHandler(userRepo *models.UserRepository, tokenRepo *models.TokenRepository) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo, TokenRepo: tokenRepo}
}
```
### 🗂️Функция ```func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)```
Запрос ``` POST http://localhost:8080/api/v1/register```
Регистрирует пользователя.
Ожидаемое тело запроса:
```json
{
  "username": "admin",
  "password": "admin123"
}
```
Логика работы:
1) Считывается json
2) Валидируется логин (от 3 до 20 символов) и пароль (От 6 символов + Доступные символы: латиница, цифры, _)
3) Проверяется существование пользователя, если уже есть такой username, то выведет ```"User already exists"```
4) Пароль хэшируется (в БД хранится только хэш пароля)
5) Если все условия успешно выполнены, то выведет ```"User registered successfully"```

### 🗂️Функция ```func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)```
Запрос ``` POST http://localhost:8080/api/v1/login```
Авторизует пользователя и выдает пару токенов (access, refresh)
Ожидаемое тело запроса:
```json
{
  "username": "admin",
  "password": "admin123"
}
```
Логика работы:
1) Поиск в базе по username
2) Проверка хэша введенного пароля
3) Генерация Access token
4) Генерация Refresh token
5) Возвращает ```200 OK``` и JSON
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "access_token": "<jwt>",
    "role": 0
  }
}
```
### 🗂️Функция ```func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request)```
Запрос ``` POST http://localhost:8080/api/v1/refresh```
Обновляет Access JWT-token
Логика работы:
1) Ищет refresh token в cookie
2) Сравнивает токен из куки и в БД + проверяет не истек ли срок
3) Удаляет старый refresh token и генерирует новый
4) Сохраняет новый токен в БД и куки
5) Создает новый access token
6) Возвращает ```200 OK``` и JSON:
```json
{
  "code": 200,
  "message": "Token refreshed",
  "data": {
    "access_token": "<new_jwt>"
  }
}
```
### 🗂️Функция ```func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)```
Запрос ``` POST http://localhost:8080/api/v1/logout```
Выход пользователя из системы
Логика работы:
1) Извлекает refresh token из cookie
2) Если найден, то удаляет его из БД и очищает cookie
3) Возвращает ```200 OK``` и сообщение ```"Logged out"```.

#  📘 handlers/logs.go — обработчики логов приложения
В файле реализуются административные эндпоинты для просмотра и скачивания логов
### 🗂️ Функция ```func ListAllLogs(w http.ResponseWriter, r *http.Request)```
Запрос ```GET http://localhost:8080/api/v2/logs```

Возвращает список всех лог файлов проекта

Логика работы:
1) Вызывает ```utils.DiscoverLogFiles(false)``` для поиска файлов с расширением ```.log```.
2) Сортирует по дате изменения (```Modified``` — последние сверху).
3) Формирует JSON-ответ со сведениями о каждом файле:
```name``` — имя файла
```dir``` — путь к директории
```path``` — полный путь
```root``` — идентификатор источника (например, "local")
```size``` — размер в байтах
```human``` — человекочитаемый размер (KB/MB)
```modified``` — дата изменения
4) Возвращает ```200 OK``` с массивом файлов.
Пример ответа:
```json
{
  "code": 200,
  "message": "OK",
  "data": [
    {
      "name": "api.log",
      "dir": "/logs",
      "path": "/logs/api.log",
      "root": "local",
      "size": 15234,
      "human": "15 KB",
      "modified": "2025-10-23 14:01:32"
    }
  ]
}
```

### 🗂️ Функция ```func DownloadAllLogs(w http.ResponseWriter, r *http.Request)```
Запрос ```GET http://localhost:8080/api/v2/logs/download-all``` 

Создает ZIP-архив со всеми логами
Логика работы:
1) Получает список файлов через ```utils.DiscoverLogFiles(true)```.
Если файлов нет → возвращает ```404 no logs found```.

2) Устанавливает заголовки:
```
Content-Type: application/zip  
Content-Disposition: attachment; filename="logs_all_<timestamp>.zip"
```

3) Архивирует файлы “на лету” через ```zip.Writer``` и ```io.Pipe()```.

4) Каждому файлу внутри архива присваивает подпапку по имени источника (```root```).

Результат:

Скачиваемый архив:
```
logs_all_20251023T150405.zip
 ├── local/
 │   ├── api.log
 │   ├── Modbus_BEMP.log
```
Такой подход исключает возможные коллизии в случае если в разных дирректориях файлы имеют одинаковое название.

### 🗂️ Функция ```func TailUnified(w http.ResponseWriter, r *http.Request)```
Пример запроса ```GET http://localhost:8080/api/v2/logs/tail?name=IRZ_ModbusOvenOk.20250919T073501.log&root=local&lines=200&format=raw```

Возвращает последние ```N``` строк указанного лог-файла

| Параметр | Обязательный | Описание                                             |
| -------- | ------------ | -----------------------------------------------------|
| `name`   | ✅           |  Имя лог-файла (`api.log`)                           |
| `lines`  | ❌           |  Количество строк с конца (по умолчанию 200)         |
| `format` | ❌           |  `json` или `raw`                                    |
| `root`   | ✅           |  Идентификатор источника логов (например, `"local"`) |

Логика работы:
1) Проверяет наличие параметров ```name``` и ```root```.

2) Находит файл через ```utils.ResolveOneByName```.

3) Открывает файл безопасно (```utils.OpenSafe```).

4) Читает последние ```N``` строк (```utils.ReadTailLines```).

5) Форматирует ответ:

```format=raw``` → возвращает обычный текст (text/plain);

```format=json``` → парсит JSON-записи или строки в скобках (```utils.ParseBracketLine```).

Пример ответа (```format=json```):
```json
{
  "code": 200,
  "message": "OK",
  "data": [
    {"level":"info","msg":"Admin requested user list","time":"2025-10-23T14:00:00Z"},
    {"level":"error","msg":"Database error","time":"2025-10-23T14:01:00Z"}
  ]
}
```

### 🗂️ Функция ```func DownloadSelectedLogs(w http.ResponseWriter, r *http.Request)```
Запрос ```GET POST http://localhost:8080/api/v2/logs/download```

Позволяет скачать выбранные логи в виде ZIP-архива.
Тело запроса:
```json
{
  "files": [
    {"name": "api.log", "root": "local"},
    {"name": "Modbus_BEMP.log", "root": "local"}
  ]
}
```
Логика работы:
1) Декодирует JSON, проверяет, что каждый элемент имеет ```name``` и ```root```.

2) Разрешает пути через ```utils.ResolveOneByName```.

3) Если не найден → ```404 not found: <file> (<root>)```.

4) Формирует ZIP-архив с выбранными файлами.

5) Устанавливает заголовки:
```
Content-Type: application/zip
Content-Disposition: attachment; filename="logs_selected_<timestamp>.zip"
```
6) Архивирует файлы “на лету” через ```addFileToZipWithRoot```.

###  🧰 Вспомогательные функции
```addFileToZipWithRoot(zw *zip.Writer, fullPath, root string) error```

Добавляет файл в ZIP в подпапку с именем источника (```root```).
Пример: ```local/api.log```.
Настраивает права доступа ```0o644``` и сохраняет дату изменения.

```parseIntDefault(s string, d int) int```

Парсит строку в число; при ошибке возвращает значение по умолчанию (```d```).


#  🔒 internal/middleware/auth.go — middleware для аутентификации и авторизации
Модуль реализует промежуточные обработчики (middleware) для проверки JWT-токена и роли пользователя.
Используется в маршрутах /api/v1 и /api/v2 для защиты эндпоинтов и разграничения прав доступа.

###  ⚙️ Основные задачи
| Middleware           | Назначение                                                                    |
| -------------------- | ----------------------------------------------------------------------------- |
| `AuthMiddleware`     | Проверяет наличие и корректность JWT-токена (`Authorization: Bearer <token>`) |
| `RoleMiddleware`     | Проверяет, что у пользователя достаточная роль для доступа к ресурсу          |
| `GetUserFromContext` | Извлекает информацию о пользователе (claims) из контекста запроса             |

### 🗂️ Функция ```func AuthMiddleware(next http.Handler) http.Handler```
Проверяет наличие и валидность JWT access-токена в каждом запросе.

Логика работы:
1) Ищет заголовок ```Authorization```.
Отсутствует → возвращает ```401 Authorization header required```.

2) Проверяет формат:
```
Authorization: Bearer <token>
```

Неверный формат → ```401 Invalid authorization format```.

3) Валидирует токен:
```go
claims, err := utils.ValidateToken(tokenString)
```

Ошибка проверки подписи или истечения срока → ```401 Invalid token```.

4) Если токен валиден — сохраняет ```claims``` (данные пользователя) в контекст:
```go
ctx := context.WithValue(r.Context(), UserContextKey, claims)
```

5) Передаёт запрос дальше по цепочке с обновлённым контекстом.

Пример ошибки:
```
{"code": 401, "message": "Invalid token"}
```

Пример использования:
```go
r.Route("/api/v1", func(r chi.Router) {
    r.Use(middleware.AuthMiddleware)
    r.Get("/profile", handlers.GetProfile)
})
```

### 🗂️ Функция ```func RoleMiddleware(requiredRole int) func(http.Handler) http.Handler```
Проверяет роль пользователя, добавленную в контекст ```AuthMiddleware```.
Используется для ограничения доступа (например, только для админов).

Логика работы:
1) Извлекает ```claims``` из контекста:
```go
claims, ok := r.Context().Value(UserContextKey).(*utils.Claims)
```

Если отсутствуют → ```401 Authentication required```.

2) Сравнивает роль пользователя с требуемой:
```go
if claims.Role < requiredRole
```

Недостаточно прав → ```403 Insufficient permissions```.

3) При успехе передаёт запрос дальше.

Пример использования:
```go
r.Route("/api/v2", func(r chi.Router) {
    r.Use(middleware.AuthMiddleware)
    r.With(middleware.RoleMiddleware(1)).Group(func(r chi.Router) {
        r.Get("/admin/users", handlers.ListUsers)
    })
})
```

🧭 Здесь 1 — это минимальная роль администратора (role=1).

Пример ошибок:
```json
{"code": 401, "message": "Authentication required"}
```

### 🗂️ Функция ```func GetUserFromContext(ctx context.Context) *utils.Claims```

Утилита для безопасного извлечения пользователя из контекста внутри обработчиков.

Пример:
```go
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    if user == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    fmt.Fprintf(w, "Hello, %s!", user.Username)
}
```

🧠 Что такое utils.Claims

Структура Claims описывает содержимое JWT-токена (передаётся из utils.jwt.go):
```go
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}
```

# 📁 Репозитории для взаимодействия с БД internal/models
## 🧩 internal/models/token_repository.go — репозиторий для работы с refresh-токенами
Отвечает за создание, чтение и удаление refresh-токенов в базе данных.
Используется в обработчиках аутентификации (handlers/auth.go) для управления сессиями пользователей.
### ⚙️ Структура
```go
type TokenRepository struct {
	DB *sql.DB
}
```

Хранит ссылку на открытое соединение с базой данных (```*sql.DB```).
Позволяет выполнять SQL-запросы для таблицы refresh_tokens.

Создаётся через конструктор:
```go
func NewTokenRepository(db *sql.DB) *TokenRepository
```
### 🗄️ Ожидаемая структура таблицы ```refresh_tokens```
```sql
CREATE TABLE refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
### 🧠 Принцип работы в системе
| Этап          | Метод                                                         | Действие                                         |
| ------------- | ------------------------------------------------------------- | ------------------------------------------------ |
| 🔑 Логин      | `SaveRefreshToken`                                            | Создаёт новый refresh-токен и сохраняет его      |
| 🔄 Обновление | `GetRefreshToken` → `DeleteRefreshToken` → `SaveRefreshToken` | Проверяет токен и заменяет его новым             |
| 🚪 Выход      | `DeleteAllForUser`                                            | Удаляет все активные refresh-токены пользователя |

## 👤 internal/models/user.go — репозиторий пользователей
Реализует модель пользователя и набор методов для работы с таблицей users в базе данных.
Используется в аутентификации (```handlers/auth.go```) и административных функциях (```handlers/admin.go```).

### ⚙️ Структура пользователя
```go
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`       // не возвращается в JSON
	Role         int       `json:"role"`    // 0=user, 1=admin
	CreatedAt    time.Time `json:"created_at"`
}
```

### 🧱 Ожидаемая структура таблицы users
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
### 🔐 Роли пользователей
| Значение            | Назначение                                      |
| ------------------- | ----------------------------------------------- |
| `0`                 | Обычный пользователь                            |
| `1`                 | Администратор                                   |
| *(опционально)* `2` | Расширенный доступ (при необходимости добавить) |

### 🧠 Принцип работы в системе
| Метод               | Назначение                         | Где используется              |
| ------------------- | ---------------------------------- | ----------------------------- |
| `CreateUser`        | Добавить пользователя              | `Register`                    |
| `GetUserByUsername` | Найти по имени                     | `Login`                       |
| `GetUserByID`       | Найти по ID                        | `Refresh`                     |
| `UserExists`        | Проверить дубликат                 | `Register`                    |
| `AdminExists`       | Проверить наличие админа           | инициализация системы         |
| `GetAllUsers`       | Получить список всех пользователей | `AdminHandler.ListUsers`      |
| `UpdateUserRole`    | Изменить роль                      | `AdminHandler.UpdateUserRole` |
