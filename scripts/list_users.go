package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type User struct {
	ID        int64
	Username  string
	Role      int
	CreatedAt string
}

func main() {
	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

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

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}
		users = append(users, user)
	}

	for _, user := range users {
		roleStr := "User"
		if user.Role == 1 {
			roleStr = "Admin"
		}
		fmt.Printf("%d | %s | %s | %s\n", user.ID, user.Username, roleStr, user.CreatedAt)
	}

	fmt.Printf("\nTotal users: %d\n", len(users))
}
