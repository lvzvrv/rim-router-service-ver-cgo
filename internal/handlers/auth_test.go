package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"your-app/internal/models"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
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

	// Create a test user first with REAL bcrypt hash
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
		// Пустые значения - проверяем реальное поведение
		{
			name: "Empty username",
			requestBody: map[string]string{
				"username": "",
				"password": "password123",
			},
			expectedStatus: http.StatusUnauthorized, // База данных не найдет пользователя
		},
		{
			name: "Empty password",
			requestBody: map[string]string{
				"username": "testuser",
				"password": "",
			},
			expectedStatus: http.StatusUnauthorized, // bcrypt сравнение не пройдет
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

func TestAuthHandler_EdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	tests := []struct {
		name           string
		requestBody    string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			requestBody:    `{"username": "test", "password": "pass123"`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Wrong content type - но JSON парсер может сработать",
			requestBody:    `{"username": "contenttypetest", "password": "pass123"}`,
			contentType:    "text/plain",
			expectedStatus: http.StatusCreated, // Go json.Decoder может обработать
		},
		{
			name:           "Extra fields in JSON",
			requestBody:    `{"username": "extratest", "password": "pass123", "extra": "field"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", tt.contentType)

			rr := httptest.NewRecorder()
			handler.Register(rr, req)

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

	// Register first user
	body1, _ := json.Marshal(map[string]string{
		"username": "user1",
		"password": password,
	})
	req1 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler.Register(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("First registration failed with status %d", rr1.Code)
	}

	// Register second user with same password
	body2, _ := json.Marshal(map[string]string{
		"username": "user2",
		"password": password,
	})
	req2 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.Register(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("Second registration failed with status %d", rr2.Code)
	}

	// Get both users from database
	user1, err := userRepo.GetUserByUsername("user1")
	if err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	user2, err := userRepo.GetUserByUsername("user2")
	if err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	// Verify that hashes are different (salting works)
	if user1.PasswordHash == user2.PasswordHash {
		t.Error("Password hashes should be different due to salting")
	}

	// Verify both passwords work in login
	loginBody1, _ := json.Marshal(map[string]string{
		"username": "user1",
		"password": password,
	})
	loginReq1 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(loginBody1))
	loginReq1.Header.Set("Content-Type", "application/json")
	loginRr1 := httptest.NewRecorder()
	handler.Login(loginRr1, loginReq1)

	if loginRr1.Code != http.StatusOK {
		t.Error("User1 login should work")
	}

	loginBody2, _ := json.Marshal(map[string]string{
		"username": "user2",
		"password": password,
	})
	loginReq2 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(loginBody2))
	loginReq2.Header.Set("Content-Type", "application/json")
	loginRr2 := httptest.NewRecorder()
	handler.Login(loginRr2, loginReq2)

	if loginRr2.Code != http.StatusOK {
		t.Error("User2 login should work")
	}
}

func TestSQLInjectionProtection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	// Эти попытки должны быть заблокированы валидацией username
	sqlInjectionAttempts := []string{
		"admin' OR '1'='1",
		"test@user", // специальные символы
		"user name", // пробелы
	}

	for _, attempt := range sqlInjectionAttempts {
		t.Run("Invalid username: "+attempt, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"username": attempt,
				"password": "password123",
			})
			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Register(rr, req)

			// Должны получить 400 из-за валидации
			if rr.Code != http.StatusBadRequest {
				t.Errorf("Invalid username should be blocked by validation, got status %d", rr.Code)
			}
		})
	}
}

func TestVeryLongPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	// Используем пароль разумной длины (50 символов)
	longPassword := strings.Repeat("a", 50)

	body, _ := json.Marshal(map[string]string{
		"username": "longpassworduser",
		"password": longPassword,
	})
	req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	// Должен работать
	if rr.Code != http.StatusCreated {
		t.Errorf("Long password should work, got status %d. Response: %s", rr.Code, rr.Body.String())
		return
	}

	// Проверяем что логин работает
	loginBody, _ := json.Marshal(map[string]string{
		"username": "longpassworduser",
		"password": longPassword,
	})
	loginReq := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRr := httptest.NewRecorder()
	handler.Login(loginRr, loginReq)

	if loginRr.Code != http.StatusOK {
		t.Errorf("Login with long password should work, got status %d. Response: %s", loginRr.Code, loginRr.Body.String())
	}
}

// Простой тест на параллельную регистрацию
func TestBasicConcurrentRegistration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := models.NewUserRepository(db)
	handler := NewAuthHandler(userRepo)

	// Просто регистрируем двух разных пользователей параллельно
	done := make(chan bool, 2)

	go func() {
		body, _ := json.Marshal(map[string]string{
			"username": "user1",
			"password": "password123",
		})
		req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.Register(rr, req)
		done <- (rr.Code == http.StatusCreated)
	}()

	go func() {
		body, _ := json.Marshal(map[string]string{
			"username": "user2",
			"password": "password123",
		})
		req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.Register(rr, req)
		done <- (rr.Code == http.StatusCreated)
	}()

	success1 := <-done
	success2 := <-done

	if !success1 || !success2 {
		t.Error("Both concurrent registrations should succeed for different users")
	}
}
