package utils

import (
	"fmt"

	"foro-unsaac-backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the computation cost for bcrypt (higher = slower but more secure)
// Production recommendation: 12-14. For tests keep at 12.
const bcryptCost = 12

type passwordService struct{}

// NewPasswordService creates a new password service
// Returns the domain interface, not the concrete struct
func NewPasswordService() domain.PasswordService {
	return &passwordService{}
}

// Hash generates a bcrypt hash from a plaintext password
// The salt is randomly generated and embedded in the hash
// Can be safely stored in the database
// Implements domain.PasswordService
func (s *passwordService) Hash(password string) (string, error) {
	// Validate input
	if password == "" {
		return "", fmt.Errorf("password cannot be empty: %w", domain.ErrValidation)
	}

	if len(password) < 8 {
		return "", fmt.Errorf("password too short: %w", domain.ErrValidation)
	}

	// Generate bcrypt hash
	// bcrypt automatically handles salt generation
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hash), nil
}

// Verify compares a bcrypt hash against a plaintext password
// Returns true if they match, false otherwise
// Safe against timing attacks (bcrypt uses constant-time comparison)
// Implements domain.PasswordService
func (s *passwordService) Verify(hash, password string) bool {
	// Don't verify empty inputs
	if hash == "" || password == "" {
		return false
	}

	// bcrypt.CompareHashAndPassword uses constant-time comparison
	// This prevents timing attacks where attacker could deduce password length
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
