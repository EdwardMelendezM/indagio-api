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

	// SelectedBorderID is the persisted FK to avatar_borders.id.
	// SelectedBorder is hydrated in any query that LEFT JOINs
	// avatar_borders (feed queries, /me, /users/:id, etc.). It is
	// NOT persisted and is not part of the users table.
	SelectedBorderID *uuid.UUID
	SelectedBorder   *AvatarBorder
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

	// SelectedBorderID is read by FindByEmail for the login flow when
	// we want to enrich the AuthResponse. The internal view is
	// otherwise unchanged; SelectedBorder is not hydrated here (the
	// auth path doesn't render the border).
	SelectedBorderID *uuid.UUID
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
