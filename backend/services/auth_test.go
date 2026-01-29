package services

import (
	"testing"

	"github.com/codyseavey/bets/config"
	"github.com/codyseavey/bets/models"
)

func setupAuthService(t *testing.T) *AuthService {
	t.Helper()
	db := setupTestDB(t)
	cfg := &config.Config{
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		JWTSecret:          "test-jwt-secret-that-is-long-enough",
		BaseURL:            "http://localhost:8080",
	}
	return NewAuthService(db, cfg)
}

func TestRegister(t *testing.T) {
	svc := setupAuthService(t)

	user, err := svc.Register("alice@example.com", "password123", "Alice")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got '%s'", user.Email)
	}
	if user.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", user.Name)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}
	if user.GoogleID != nil {
		t.Error("expected nil GoogleID for local user")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := setupAuthService(t)

	if _, err := svc.Register("alice@example.com", "password123", "Alice"); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err := svc.Register("alice@example.com", "differentpass", "Alice2")
	if err == nil {
		t.Fatal("expected error for duplicate email registration")
	}
}

func TestLogin(t *testing.T) {
	svc := setupAuthService(t)

	if _, err := svc.Register("bob@example.com", "securepass1", "Bob"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	user, err := svc.Login("bob@example.com", "securepass1")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if user.Email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', got '%s'", user.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := setupAuthService(t)

	if _, err := svc.Register("bob@example.com", "securepass1", "Bob"); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err := svc.Login("bob@example.com", "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	svc := setupAuthService(t)

	_, err := svc.Login("nobody@example.com", "password")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestLogin_GoogleOnlyUser(t *testing.T) {
	svc := setupAuthService(t)

	// Create a Google-only user directly in the DB
	googleID := "google-123"
	googleUser := &models.User{
		ID:       "guser1",
		GoogleID: &googleID,
		Email:    "google@example.com",
		Name:     "Google User",
	}
	if err := svc.db.Create(googleUser).Error; err != nil {
		t.Fatalf("failed to create Google user: %v", err)
	}

	_, err := svc.Login("google@example.com", "anypassword")
	if err == nil {
		t.Fatal("expected error when logging in to a Google-only account with password")
	}
}

func TestRegister_JWTRoundTrip(t *testing.T) {
	svc := setupAuthService(t)

	user, err := svc.Register("jwt@example.com", "password123", "JWT User")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	token, err := svc.GenerateJWT(user)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	claims, err := svc.ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID '%s', got '%s'", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Errorf("expected email '%s', got '%s'", user.Email, claims.Email)
	}
}
