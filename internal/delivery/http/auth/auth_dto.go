package auth

// RegisterRequest DTO
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=80"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// VerifyOTPRequest DTO
type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

// LoginRequest DTO
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ForgotPasswordRequest DTO
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest DTO
type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UserResponse DTO
type UserResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Blocked   bool       `json:"blocked"`
	AvatarURL *string    `json:"avatar_url"`
	Border    *BorderDTO `json:"border,omitempty"` // nil when no border selected
	CreatedAt string     `json:"created_at"`
}

// BorderDTO is the small subset of an avatar border embedded inside
// a UserResponse. We don't reuse the full catalog response (no
// is_unlocked, no is_selected, no sort_order) because the consumer
// of an "author" object only needs the asset to render the border
// over the avatar. Keeping it in the auth package avoids a circular
// import (auth → avatar_border → auth).
type BorderDTO struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	AssetURL     string `json:"asset_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Tier         string `json:"tier"`
}

// AuthResponse DTO
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}
