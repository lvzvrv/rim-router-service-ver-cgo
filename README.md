#  ⚙️ Документация проекта 

## 🧰 Сборка и запуск проекта rim-router-service-ver-cgo

Файл автоматизирует сборку, сжатие, тестирование и очистку проекта.
Ниже описано, какие инструменты необходимы для успешной компиляции и как их установить.

---

### ⚙️ Требуемое окружение

Для сборки проекта необходимо наличие следующих компонентов:

#### 1. **Go (Golang)**

Используется для компиляции исходного кода.

* Минимальная версия: **Go 1.21+**
* Проверка установки:

  ```bash
  go version
  ```
* Установка (Linux / macOS):

  ```bash
  sudo apt install golang-go -y       # Debian/Ubuntu
  # или
  brew install go                     # macOS
  ```
* Установка вручную (если требуется конкретная версия):

  ```bash
  wget https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  ```

#### 2. **UPX (Ultimate Packer for eXecutables)**

Необходим для сжатия бинарников после сборки (опционально, используется в команде `make upx`).

* Проверка установки:

  ```bash
  upx --version
  ```
* Установка:

  ```bash
  sudo apt install upx -y             # Debian/Ubuntu
  # или
  brew install upx                    # macOS
  ```

#### 3. **CGO и компилятор C (для Linux-сборки)**

При кросс-компиляции (`make linux`) требуется поддержка **CGO**.

* Установите инструменты сборки:

  ```bash
  sudo apt install build-essential -y
  ```

---

### 🧱 Основные команды Makefile

| Команда      | Описание                                                                            |
| ------------ | ----------------------------------------------------------------------------------- |
| `make build` | Собирает оптимизированный бинарник (без отладочной информации, минимальный размер). |
| `make upx`   | Сжимает бинарник с помощью UPX (`--best --lzma`).                                   |
| `make run`   | Запускает сервер напрямую через `go run`.                                           |
| `make test`  | Запускает все тесты проекта.                                                        |
| `make clean` | Удаляет директории `build/` и `logs/`.                                              |
| `make linux` | Кросс-компиляция под Linux `amd64` с поддержкой CGO.                                |

---

### 🧩 Пример полного цикла сборки

```bash
# 1. Клонируем репозиторий
git clone https://github.com/lvzvrv/rim-router-service-ver-cgo.git
cd rim-router-service-ver-cgo

# 2. Устанавливаем зависимости
sudo apt install golang-go upx build-essential -y

# 3. Собираем проект
make build

# 4. (Необязательно) Сжимаем бинарник
make upx

# 5. Запускаем
make run
```

---

### ✅ Результат сборки

После выполнения `make build` создаётся бинарник:

```
/build/router-service
```

или (при `make linux`):

```
/build/router-service-linux
```

Можно проверить размер и запустить вручную:

```bash
ls -lh build/
./build/router-service
```

---


# 📁 Структура проекта
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

### 🗂️ Функция runMigrations()
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

###  🗂️ Функция ```func NewAdminHandler(userRepo *models.UserRepository) *AdminHandler```

Конструктор обработчика. Создает новый экзмепляр AdminHandler, настраивая логирование.


###  🗂️ Функция ```func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request)```

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


###  🗂️ Функция ```func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request)```

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
### 🗂️ Функция ```func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)```
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

### 🗂️ Функция ```func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)```
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
### 🗂️ Функция ```func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request)```
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
### 🗂️ Функция ```func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)```
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


# 📁 internal/utils — вспомогательные утилиты проекта
Папка utils содержит вспомогательные модули, которые обеспечивают общие функции проекта — от логирования и работы с файлами до генерации и проверки JWT-токенов.
Эти утилиты используются в разных частях системы (handlers, middleware, models) и служат “служебным слоем” для переиспользуемой логики.

## 📦 Основная структура
| Файл               | Назначение                                                                                                                                              |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`jwt.go`**       | Работа с токенами авторизации (генерация и валидация JWT, создание secure refresh-токенов).                                                             |
| **`logfinder.go`** | Поиск и сбор лог-файлов на диске, включая фильтрацию, информацию о размере, дате изменения и “корневом источнике” (`root`).                             |
| **`logging.go`**   | Управление системными логами приложения — определение путей к файлам логов, инициализация логеров (`zerolog`), создание директорий для хранения логов.  |
| **`logparser.go`** | Разбор и нормализация строк логов: парсинг JSON-записей, форматирование строк с квадратными скобками, извлечение временных меток и уровней логирования. |

## 🔐 internal/utils/jwt.go — работа с токенами (JWT и refresh)
Файл отвечает за генерацию и валидацию JWT access-токенов, а также за создание безопасных refresh-токенов.

### Данные, которые шифруются и проверяются в каждом запросе
```go
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}
```

### 🧾 `func GenerateAccessToken(user *models.User) (string, error)`
Создаёт **JWT access-токен** для авторизации.
#### Алгоритм работы:
1. Загружает настройки из `config.GetJWTConfig()` (секрет, время жизни и т.д.).
2. Формирует объект `Claims` с данными пользователя.
3. Устанавливает время:
   - `ExpiresAt` — срок действия токена (например, 15 минут),
   - `IssuedAt` — время создания.
4. Создаёт токен с подписью HMAC-SHA256:
   ```go
   token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

5. Подписывает токен секретным ключом:
	```go
   token.SignedString([]byte(jwtConfig.Secret))
	```
6. Возвращает готовый JWT-токен (строкой).
Используется в:
`AuthHandler.Login()` — при успешном входе;
`AuthHandler.Refresh()` — при обновлении токенов.
Пример содержимого JWT (декодированный payload):
	```go
	{
  	"user_id": 1,
  	"username": "admin",
  	"role": 1,
  	"exp": 1730000000,
  	"iat": 1729996400
	}
	```
## ✅ func ValidateToken(tokenString string) (*Claims, error)
Проверяет JWT-токен на корректность и срок действия.
#### Алгоритм работы:
1. Загружает Secret из конфига.
2. Разбирает токен:
	```go
	jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
	    return []byte(jwtConfig.Secret), nil
	})
	```
3. Проверяет:
подпись (`signature`),
срок действия (`exp`),
формат токена.
4. Возвращает:
`*Claims` — данные пользователя из токена;
`error` — если токен невалиден (неверная подпись, истёк, повреждён и т.д.).
Используется в:
`middleware.AuthMiddleware()` — при каждом защищённом запросе.

## 🔒 func GenerateSecureToken() (string, error)
Создаёт refresh-токен — случайную криптографически стойкую строку длиной 64 символа (256 бит энтропии).
Этот токен не является JWT и хранится в базе данных.

#### Логика:

1. Создаёт буфер 32 байта:
```go
buf := make([]byte, 32)
```
2. Заполняет случайными байтами (crypto/rand).
3. Преобразует в hex-строку (удобно хранить в БД и передавать в cookie).
Пример:
`d4a3b9a50d4a5ef3d35f4f28e1a142df76cf8d6a79b87b0c8ce7c1c7f0cbba8b`
Используется в:
`AuthHandler.Login()` — при входе пользователя;
`AuthHandler.Refresh()` — при обновлении пары токенов.

## 🧭 internal/utils/logfinder.go — поиск и доступ к лог-файлам
Утилиты для обнаружения, фильтрации и безопасного открытия лог-файлов в заранее разрешённых корнях (local, sd). Работает только с файлами с расширением .log.

Ищет только .log (регистронезависимо).
```go
var extLogRe = regexp.MustCompile(`(?i)\.log$`)
```
📦 Структуры данных
```go
type LogInfo struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	RootID   string    `json:"root_id"` // "local" / "sd"
}

type Root struct {
	ID   string // "local", "sd"
	Path string // абсолютный путь к папке tir_logs
}
```
### 🌱 func ListRoots() []Root
Возвращает список корней, где могут лежать логи.

#### Логика:
1. Добавляет `local: ./tir_logs` рядом с исполняемым файлом (создаёт при необходимости `0o755`).
2. Рекурсивно обходит `/mnt` и добавляет все каталоги, заканчивающиеся на `tir_logs` как `sd`.
3. Возвращает отсортированный список (стабильный порядок по `Path`).

### 📃 func AllowedRoots() []string
Совместимая обёртка над ListRoots() — возвращает только пути корней. Используется для checks в OpenSafe.

### 🧪 func LooksLikeLog(name string) bool
Проверяет, что имя выглядит как лог-файл (*.log), без учёта регистра.

### 🛡️ func WithinAllowedRoots(p string, roots []string) bool
Проверяет, что абсолютный путь p лежит внутри одного из разрешённых корней.

### 🧰 func strconvI(i int) string
Мини-утилита для конвертации положительного int в строку без подключения strconv.

### 📚 func DiscoverLogFiles(includeTimestamped bool) ([]LogInfo, error)
Ищет все .log во всех корнях из ListRoots(), собирая метаданные.
#### Логика:
1. Для каждого `root.Path` выполняет `filepath.WalkDir`.
Пропускает директории; берёт только файлы, прошедшие `LooksLikeLog`.
2. Для каждого файла заполняет `LogInfo (Abs, Dir, Size, ModTime, RootID)`.
3. Возвращает срез найденных логов.

### 🔍 func FindLogsByName(name string) ([]LogInfo, error)
Находит все совпадения по имени файла среди допустимых корней.
#### Логика:
1. Валидирует `name` (не пустой и *.log).
2. Берёт все логи через `DiscoverLogFiles(true)`.
3. Фильтрует по имени без учета регистра (`EqualFold`).
4. Сортирует стабильно:
Сначала `local`, затем `sd`;
Внутри одного RootID — по времени изменения (сначала более новые).

### 🎯 func ResolveOneByName(name, rootHint string) (LogInfo, error)
Возвращает ровно один файл по имени и явному указанию корня.
#### Логика:
1. Вызывает `FindLogsByName(name)`.
2. Требует непустой `rootHint` (иначе ошибка).
3. Возвращает первый `LogInfo` с совпадающим `RootID`.
4. Если не найдено — соответствующая ошибка (`"log not found", "no match for given root"`).

### 🔓 func OpenSafe(path string) (*os.File, error)
Безопасно открывает файл только если он лежит в разрешённых корнях.
#### Логика:
1. Получает список корней: `AllowedRoots()`.
2. Проверяет `WithinAllowedRoots(path, roots)`.
3. Если ок — `os.Open(path)`, иначе ошибка `"path not allowed"`.

🧪 Пример результата LogInfo (JSON)
```json
{
  "path": "/opt/app/tir_logs/api.log",
  "name": "api.log",
  "dir": "/opt/app/tir_logs",
  "size": 15234,
  "modified": "2025-10-23T14:01:32Z",
  "root_id": "local"
}
```

## ⚙️ internal/utils/log_utils.go — система логирования и ротации логов

Файл отвечает за выбор директории для хранения логов, автоматическую ротацию при превышении размера, контроль свободного места и удаление старых логов.

---

### 📁 Константы конфигурации

```go
const (
	PreferredSDPath  = "/mnt"       // путь, где ищется папка tir_logs на SD-карте
	LocalLogPath     = "./tir_logs" // fallback — локальная директория
	LogFileName      = "api.log"    // имя основного лог-файла
	MaxLogSizeBytes  = 5 * 1024 * 1024 // максимальный размер (5 MB)
	MaxArchivedFiles = 5               // хранится не более 5 архивов
	MinFreeSpaceMB   = 6               // минимум свободного места в MB
)
```

---

### 📂 `func ChooseLogDir() string`

Выбирает место для хранения логов:

1. Проверяет наличие папки `tir_logs` на SD-карте (`/mnt`).
2. Если она существует — использует её.
3. Если нет — создаёт локальную `./tir_logs` рядом с бинарником.

📄 Использует вспомогательную функцию `ensureDir()` для проверки прав записи.

---

### 🧩 `func ensureDir(dir string) error`

Создаёт директорию (если нет) и проверяет возможность записи, создавая и удаляя тестовый файл `.test`.

---

### 📌 `func LogDir() string`

Возвращает активный путь к директории логов. Если не задан — вызывает `ChooseLogDir()`.

---

### 📄 `func LogFilePath() string`

Возвращает абсолютный путь к файлу `api.log` в активной директории логов.

---

### 🪣 Структура `RotatingWriter`

Реализует собственный логгер с автоматической ротацией файлов при превышении размера или недостатке места на диске.

```go
type RotatingWriter struct {
	file *os.File
}
```

#### 🧾 `func NewRotatingWriter() (*RotatingWriter, error)`

Создаёт и открывает файл `api.log` для записи. Если его нет — создаёт новый.

#### ✍️ `func (w *RotatingWriter) Write(p []byte)`

Основной метод записи лога. Алгоритм:

1. Проверяет размер текущего файла.
2. Проверяет доступное место на диске (`checkDiskSpaceAndCleanup()`).
3. Если файл превышает `MaxLogSizeBytes` — вызывает `rotate()`.
4. Записывает данные в файл.

#### 🔁 `func (w *RotatingWriter) rotate() error`

Реализует ротацию логов:

1. Закрывает текущий файл.
2. Переименовывает `api.log` в `api.YYYYMMDDTHHMMSS.log`.
3. Создаёт новый `api.log`.
4. Логирует успешную ротацию и вызывает `cleanupOldLogs()` для удаления старых архивов.

#### 🧹 `func (w *RotatingWriter) Close() error`

Закрывает файл при завершении работы.

---

### 🗑️ `func cleanupOldLogs()`

Удаляет старые архивы логов по шаблону `api.YYYYMMDDTHHMMSS.log`, оставляя только последние `MaxArchivedFiles`.

1. Сканирует директорию логов.
2. Сортирует архивы по дате изменения.
3. Удаляет самые старые.

---

### 💾 `func checkDiskSpaceAndCleanup() error`

Проверяет свободное место на диске с помощью `syscall.Statfs`.

1. Если меньше `MinFreeSpaceMB` — предупреждает в логе и удаляет старые архивы.
2. Повторно проверяет после очистки.
3. Если всё ещё недостаточно — блокирует запись и возвращает ошибку.

---

### ⏱️ `func FormatTS(t time.Time) string`

Возвращает время в формате RFC3339 (UTC) для логов.

---

### 📏 `func HumanSize(n int64) string`

Преобразует байты в человекочитаемый формат (B, KB, MB). Пример:

```
HumanSize(2048) → "2.0 KB"
HumanSize(3145728) → "3.0 MB"
```

---

### 🧠 Итог

Модуль обеспечивает:

* автоматический выбор места хранения логов (SD или локально),
* безопасную запись с проверкой свободного места,
* автоматическую ротацию и очистку старых файлов,
* форматирование и удобное отображение размеров.

## 🧾 internal/utils/logparser.go — парсинг и чтение логов

Файл содержит утилиты для чтения последних строк лог-файлов, разбора форматированных логов и потоковой передачи строк.

---

### 📄 Общие сведения

Используется для:

* эффективного чтения последних N строк больших логов (без загрузки всего файла);
* парсинга строк формата `[timestamp] [LEVEL] message` в структуру (JSON-совместимую map);
* потокового чтения логов построчно через канал (для стриминга или live tail).

---

### ⚙️ Регулярные выражения и форматы времени

```go
var (
	bracketLineRe = regexp.MustCompile(`^\[(?P<ts>[^]]+)\]\s*\[(?P<level>[^]]+)\]\s*(?P<msg>.*)$`)
	timeAltLayout = "2006-01-02 15:04:05,000" // формат [2025-09-19 04:37:38,155]
)
```

* **`bracketLineRe`** — извлекает время, уровень логирования и сообщение из строки.
* **`timeAltLayout`** — шаблон формата времени, используемый в логах.

---

### 📜 `func ReadTailLines(f *os.File, n int) ([]string, error)`

Эффективно читает **последние N строк** из файла, начиная с конца.

#### 🔍 Алгоритм работы:

1. Получает размер файла через `f.Stat()`.
2. Считывает файл с конца блоками по 4 KB, пока не найдено N строк.
3. Разделяет результат по `\n` и возвращает только последние N.
4. Очищает пустую последнюю строку, если она есть.

💡 *Используется для отображения последних строк логов (например, API или Modbus) без чтения всего файла.*

---

### 🧩 `func ParseBracketLine(s string) map[string]any`

Парсит строку формата:

```
[2025-09-19 04:37:38,155] [ERROR] ModbusServiceFunctions::ReadInt: Read timeout
```

и преобразует её в JSON-совместимую структуру:

```json
{
  "time": "2025-09-19T04:37:38.155Z",
  "level": "error",
  "module": "ModbusServiceFunctions",
  "message": "Read timeout",
  "raw": "[2025-09-19 04:37:38,155] [ERROR] ModbusServiceFunctions::ReadInt: Read timeout"
}
```

#### 🔍 Алгоритм разбора:

1. Извлекает поля `ts`, `level`, `msg` с помощью `bracketLineRe`.
2. Конвертирует `ts` из формата `2006-01-02 15:04:05,000` в `RFC3339Nano`.
3. Разделяет `msg` по `::` и `:` для выделения `module` и очищенного `message`.
4. Возвращает итоговую структуру.

Если строка не соответствует шаблону, возвращает `map[string]any{"raw": s}`.

---

### 🔄 `func StreamLines(r io.Reader, ch chan<- string)`

Читает входной поток (`io.Reader`) построчно и отправляет каждую строку в канал `ch`.

#### Принцип работы:

1. Создаёт `bufio.Scanner` для построчного чтения.
2. Передаёт каждую строку в канал.
3. После завершения чтения — закрывает канал.

💡 *Используется для потоковой передачи логов (например, при live-трансляции логов в веб-интерфейсе или CLI).*
