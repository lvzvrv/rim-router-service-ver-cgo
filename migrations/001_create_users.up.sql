CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role INTEGER NOT NULL DEFAULT 0, -- 0=user, 1=admin
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Создаем индекс для быстрого поиска по username
CREATE INDEX idx_users_username ON users(username);