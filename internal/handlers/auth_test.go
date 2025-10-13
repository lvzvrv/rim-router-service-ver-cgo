package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	database "your-app/internal/db"
	"your-app/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

// Helper to create a real bcrypt hash for testing
func createPasswordHash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func TestAuthHandler_Register(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Successful registration",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "password123",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Username too short",
			requestBody: map[string]string{
				"username": "ab",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Username must be 3-20 characters",
		},
		{
			name: "Password too short",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "12345",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password must be at least 6 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Register(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &response)
				if response["message"] != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, response["message"])
				}
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	realHash, err := createPasswordHash("password123")
	if err != nil {
		t.Fatalf("Failed to create password hash: %v", err)
	}

	err = userRepo.CreateUser("testuser", realHash, 0)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
	}{
		{
			name: "Successful login",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "password123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Wrong password",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Non-existent user",
			requestBody: map[string]string{
				"username": "nonexistent",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Login(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestPasswordHashingSecurity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	password := "samepassword123"

	// Register two users with same password
	body1, _ := json.Marshal(map[string]string{"username": "user1", "password": password})
	body2, _ := json.Marshal(map[string]string{"username": "user2", "password": password})

	req1 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body1))
	req2 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body2))
	req1.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Content-Type", "application/json")

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()
	handler.Register(rr1, req1)
	handler.Register(rr2, req2)

	if rr1.Code != http.StatusCreated || rr2.Code != http.StatusCreated {
		t.Fatal("Registration failed")
	}

	user1, _ := userRepo.GetUserByUsername("user1")
	user2, _ := userRepo.GetUserByUsername("user2")

	if user1.PasswordHash == user2.PasswordHash {
		t.Error("Hashes for identical passwords should differ (bcrypt salt missing?)")
	}
}
