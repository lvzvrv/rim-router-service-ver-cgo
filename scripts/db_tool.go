//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // Используем CGO-драйвер
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run scripts/db_tool.go list-users")
		fmt.Println("  go run scripts/db_tool.go create-admin <username> <password>")
		fmt.Println("  go run scripts/db_tool.go sql \"SELECT * FROM users\"")
		os.Exit(1)
	}

	command := os.Args[1]

	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	switch command {
	case "list-users":
		listUsers(db)
	case "create-admin":
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run scripts/db_tool.go create-admin <username> <password>")
			os.Exit(1)
		}
		createAdmin(db, os.Args[2], os.Args[3])
	case "sql":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run scripts/db_tool.go sql \"SQL_QUERY\"")
			os.Exit(1)
		}
		executeSQL(db, os.Args[2])
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}

func listUsers(db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, username, role, created_at 
		FROM users 
		ORDER BY id
	`)
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	fmt.Println("Users in database:")
	fmt.Println("ID | Username | Role | Created At")
	fmt.Println("---------------------------------")

	count := 0
	for rows.Next() {
		var id int64
		var username string
		var role int
		var createdAt string

		err := rows.Scan(&id, &username, &role, &createdAt)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		roleStr := "User"
		if role == 1 {
			roleStr = "Admin"
		}
		fmt.Printf("%d | %s | %s | %s\n", id, username, roleStr, createdAt)
		count++
	}

	fmt.Printf("\nTotal users: %d\n", count)
}

func createAdmin(db *sql.DB, username, password string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, string(hashedPassword), 1,
	)
	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	fmt.Printf("Admin user '%s' created/updated successfully\n", username)
}

func executeSQL(db *sql.DB, query string) {
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Failed to execute query:", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}

	// Заголовки
	for i, col := range columns {
		if i > 0 {
			fmt.Print(" | ")
		}
		fmt.Print(col)
	}
	fmt.Println()
	fmt.Println("---------------------------------")

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	count := 0
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}

		for i, val := range values {
			if i > 0 {
				fmt.Print(" | ")
			}
			switch v := val.(type) {
			case nil:
				fmt.Print("NULL")
			case []byte:
				fmt.Print(string(v))
			default:
				fmt.Print(v)
			}
		}
		fmt.Println()
		count++
	}

	fmt.Printf("\nTotal rows: %d\n", count)
}
