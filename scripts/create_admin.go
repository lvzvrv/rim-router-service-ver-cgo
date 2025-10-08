package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite" // Используем драйвер без CGO
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run scripts/create_admin.go <username> <password>")
		os.Exit(1)
	}

	username := os.Args[1]
	password := os.Args[2]

	// Используем modernc.org/sqlite вместо go-sqlite3
	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Создаем администратора (role = 1)
	_, err = db.Exec(
		"INSERT OR REPLACE INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, string(hashedPassword), 1,
	)

	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	fmt.Printf("Admin user '%s' created successfully\n", username)
}
