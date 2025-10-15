package db

import (
	"log"
	"os"

	"rim-router-service-ver-cgo/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// SeedAdmin проверяет, есть ли администратор; если нет — создаёт.
func SeedAdmin(userRepo *models.UserRepository) {
	exists, err := userRepo.AdminExists()
	if err != nil {
		log.Printf("❌ Failed to check admin existence: %v", err)
		return
	}

	if exists {
		log.Println("✅ Admin already exists, skipping seeding")
		return
	}

	// Читаем логин и пароль из окружения (если заданы)
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Failed to hash admin password: %v", err)
		return
	}

	if err := userRepo.CreateUser(username, string(hashed), 1); err != nil {
		log.Printf("❌ Failed to create admin: %v", err)
		return
	}

	log.Printf("✅ Admin user created successfully! username=%s, password=%s", username, password)
}
