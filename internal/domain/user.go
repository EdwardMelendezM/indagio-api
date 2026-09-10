package domain

import (
	"time"

	"github.com/google/uuid"
)

// User — domain entity (Public view, no password)
type User struct {
	ID            uuid.UUID
	Name          string
	Email         string
	Role          Role // "estudiante" or "moderador"
	Verified      bool
	AvatarURL     *string // JSON document with thumb/medium/full variants (legacy: plain URL)
	AvatarVersion int     // Bumped on every avatar change — used as cache-busting query param.
	CreatedAt     time.Time
	Blocked       bool
}

// UserInternal — private view (with password for auth layer only)
type UserInternal struct {
	ID            uuid.UUID
	Name          string
	Email         string
	Password      string // hashed
	Role          Role
	Verified      bool
	AvatarURL     *string
	AvatarVersion int
	CreatedAt     time.Time
	Blocked       bool
}

// Role is a type-safe role representation
type Role string

const (
	RoleStudent   Role = "estudiante"
	RoleModerator Role = "moderador"
)

// Validate user domain invariants
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrValidation
	}
	if u.Name == "" {
		return ErrValidation
	}
	return nil
}

// OTPCode — OTP domain entity
type OTPCode struct {
	Email     string
	Code      string
	ExpiresAt time.Time
	Consumed  bool
}
