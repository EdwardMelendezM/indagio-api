package admin_user

// AdminRequestOTPRequest is the body expected by POST /api/admin/auth/request-otp.
type AdminRequestOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// AdminLoginRequest is the body expected by POST /api/admin/auth/login.
type AdminLoginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ── Responses ───────────────────────────────────────────────────────────────

// AdminLoginResponse is returned on a successful admin login.
type AdminLoginResponse struct {
	Message string `json:"message"`
}

type VerifyAdminOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type AuthAdminResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	Admin        UserResponse `json:"user"`
}

// UserResponse DTO
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
