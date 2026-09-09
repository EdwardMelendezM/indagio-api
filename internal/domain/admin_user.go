package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AdminUser represents a registered administrator in the user_adm table.
// Admins are separate from regular users and authenticate via OTP only.
type AdminUser struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
	DeletedAt *time.Time
}

// AdminUserRepository defines the data-access contract for admin users.
// Only the innermost query needed by the usecase is exposed here.
type AdminUserRepository interface {
	// FindByEmail returns the active admin with the given email.
	// Returns ErrNotFound when no active record exists.
	FindByEmail(ctx context.Context, email string) (*AdminUser, error)

	FindById(ctx context.Context, id uuid.UUID) (*AdminUser, error)

	// MarkVerified return
	MarkVerified(ctx context.Context, email string) error

	// Create adds a new admin user to the database. This is not exposed via any public API, but can be used internally to seed the initial admin.
	Create(ctx context.Context, email string) (*AdminUser, error)

	// FindFirst Find first active
	FindFirst(ctx context.Context) (*AdminUser, error)
}

// AdminUsecase defines the business-logic contract for admin authentication.
type AdminUsecase interface {
	// RequestOTP verifies the email is a registered admin and sends a one-time code.
	// A generic error is returned regardless of whether the email exists,
	// to prevent email-enumeration attacks.
	RequestOTP(ctx context.Context, email string) error

	// Login validates the OTP and returns a signed access token with role "admin".
	Login(ctx context.Context, email string) error

	VerifyOTP(ctx context.Context, email, code string) (*AdminUser, error)

	GenerateTokens(userID uuid.UUID, role Role) (accessToken, refreshToken string, err error)

	BlockUser(ctx context.Context, id uuid.UUID) error
}
