package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository — repository interface (implemented by postgres layer)
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*UserInternal, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Create(ctx context.Context, name, email string, passwordHash string) (*User, error)
	MarkVerified(ctx context.Context, email string) error
	UpdateName(ctx context.Context, userID uuid.UUID, name string) error
	UpdatePassword(ctx context.Context, email string, passwordHash string) error
	BlockUser(ctx context.Context, id uuid.UUID) error

	// Avatar management — the avatar_url column now stores a JSON
	// document with thumb/medium/full variants. UpdateAvatarURL accepts
	// the JSON string (nil sets the column to NULL); ClearAvatar wipes
	// the avatar entirely (sets to NULL). Both atomically bump the
	// avatar_version counter used for cache-busting.
	UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarJSON *string) (int, error)
	ClearAvatar(ctx context.Context, userID uuid.UUID) (int, error)

	// UpdateSelectedBorder sets the user's selected avatar border.
	// borderID == nil clears the selection (sets the column to NULL).
	// Returns domain.ErrNotFound when the user is soft-deleted.
	UpdateSelectedBorder(ctx context.Context, userID uuid.UUID, borderID *uuid.UUID) error

	// ListAvailable returns verified, non-deleted, non-blocked users (excluding excludeUserID)
	// that the caller can start a conversation with. An empty search returns all
	// such users. Returns the requested page plus the total matching count.
	ListAvailable(ctx context.Context, excludeUserID uuid.UUID, search string, limit, offset int) ([]User, int, error)
	// ListAll returns all verified, non-deleted users (including blocked) for admin management.
	// Returns the requested page plus the total matching count.
	ListAll(ctx context.Context, search string, limit, offset int) ([]User, int, error)
}

// OTPRepository — OTP repository interface
type OTPRepository interface {
	Create(ctx context.Context, email, code string) error
	ValidateAndConsume(ctx context.Context, email, code string) (bool, error)
}

// EmailService — email service interface (implemented by utils layer)
type EmailService interface {
	SendOTP(ctx context.Context, email, code string) error
	SendPasswordReset(ctx context.Context, email, code string) error
}

// TokenService — token generation interface
type TokenService interface {
	GenerateAccessToken(userID uuid.UUID, role Role) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}

// TokenClaims — decoded token payload
type TokenClaims struct {
	UserID uuid.UUID
	Role   Role
}

// PasswordService — password service interface
type PasswordService interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

// AuthUsecase — defines all authentication business logic
type AuthUsecase interface {
	Register(ctx context.Context, name, email, password string) error
	SendOTP(ctx context.Context, email string) error
	VerifyOTP(ctx context.Context, email, code string) (*User, error)
	Login(ctx context.Context, email, password string) (*User, error)
	GenerateTokens(userID uuid.UUID, role Role) (accessToken, refreshToken string, err error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateUserName(ctx context.Context, requesterID uuid.UUID, targetUserID uuid.UUID, name string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, email, code, newPassword string) error
}

// AdminService
type AdminService interface {
	Verify(id string) bool
}
