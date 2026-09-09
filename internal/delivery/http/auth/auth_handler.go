package auth

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foro-unsaac-backend/internal/delivery/http/utils"
	"foro-unsaac-backend/internal/domain"
)

type AuthHandler struct {
	uc     domain.AuthUsecase
	logger *slog.Logger
}

func NewAuthHandler(uc domain.AuthUsecase, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{uc: uc, logger: logger}
}

// Register godoc
// @Summary		Create user account
// @Description	Register a new user with email verification via OTP
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body body RegisterRequest true "Registration data"
// @Success		201 {object} map[string]string "Verification code sent"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		422 {object} map[string]string "Validation failed"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.uc.Register(ctx, req.Name, req.Email, req.Password); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Verifica tu codigo en tu correo"})
}

// VerifyOTP godoc
// @Summary		Verify OTP and activate account
// @Description	Verify the 6-digit code sent to email to activate account
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body body VerifyOTPRequest true "Email and OTP code"
// @Success		200 {object} AuthResponse "User authenticated with tokens"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		422 {object} map[string]string "Invalid or expired OTP"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.uc.VerifyOTP(ctx, req.Email, req.Code)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	access, refresh, err := h.uc.GenerateTokens(user.ID, user.Role)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	// Set httpOnly cookie
	c.SetCookie("refresh_token", refresh, 7*24*3600, "/api/auth", "", false, true)

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         ToUserResponse(user),
	})
}

// Login godoc
// @Summary		Login user
// @Description	Authenticate user with email and password
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body body LoginRequest true "Email and password"
// @Success		200 {object} AuthResponse "User authenticated with tokens"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		401 {object} map[string]string "Invalid credentials"
// @Failure		422 {object} map[string]string "Validation failed"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.uc.Login(ctx, req.Email, req.Password)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	access, refresh, err := h.uc.GenerateTokens(user.ID, user.Role)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.SetCookie("refresh_token", refresh, 7*24*3600, "/api/auth", "", false, true)

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         ToUserResponse(user),
	})
}

// Refresh godoc
// @Summary		Refresh access token
// @Description	Generate a new access token using refresh token
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Success		200 {object} map[string]string "New access token"
// @Failure		401 {object} map[string]string "Invalid or missing refresh token"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	token, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token missing"})
		return
	}

	claims, err := h.uc.ValidateRefreshToken(token)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	// Get user to fetch role
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.uc.GetUserByID(ctx, claims.UserID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	access, _, err := h.uc.GenerateTokens(user.ID, user.Role)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": access})
}

// Logout godoc
// @Summary		Logout user
// @Description	Clear refresh token cookie to logout user
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Success		200 {object} map[string]string "Logout successful"
// @Router		/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// Me godoc
// @Summary		Get current user profile
// @Description	Retrieve authenticated user's profile information
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Success		200 {object} UserResponse "User profile"
// @Failure		401 {object} map[string]string "Authentication required"
// @Failure		404 {object} map[string]string "User not found"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/me [get]
// @Security	BearerAuth
func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		h.logger.Error("invalid user id type in context", "type", userIDVal)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := h.uc.GetUserByID(ctx, userID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, ToUserResponse(user))
}

// ForgotPassword godoc
// @Summary		Request password reset
// @Description	Send a password reset code to the user's email
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body body ForgotPasswordRequest true "Email address"
// @Success		200 {object} map[string]string "Reset code sent if email exists"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		422 {object} map[string]string "Validation failed"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.uc.ForgotPassword(ctx, req.Email); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Si el correo existe, se enviara un codigo de recuperacion"})
}

// ResetPassword godoc
// @Summary		Reset password with OTP code
// @Description	Reset user's password using the OTP code sent to their email
// @Tags		Auth
// @Accept		json
// @Produce		json
// @Param		body body ResetPasswordRequest true "Email, code, and new password"
// @Success		200 {object} map[string]string "Password reset successful"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		422 {object} map[string]string "Invalid or expired OTP"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.uc.ResetPassword(ctx, req.Email, req.Code, req.NewPassword); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contrasena actualizada correctamente"})
}

// Helpers
func ToUserResponse(u *domain.User) UserResponse {
	resp := UserResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		Email:     u.Email,
		Role:      string(u.Role),
		Blocked:   u.Blocked,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
	if u.SelectedBorder != nil {
		b := u.SelectedBorder
		resp.Border = &BorderDTO{
			ID:           b.ID.String(),
			Slug:         b.Slug,
			Name:         b.Name,
			AssetURL:     b.AssetURL,
			ThumbnailURL: b.ThumbnailURL,
			Tier:         b.Tier,
		}
	}
	return resp
}

func RegisterAuthRoutes(rg *gin.RouterGroup, h *AuthHandler, authMiddleware gin.HandlerFunc) {
	auth := rg.Group("/auth")
	auth.POST("/register", h.Register)
	auth.POST("/verify-otp", h.VerifyOTP)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/logout", h.Logout)
	auth.POST("/forgot-password", h.ForgotPassword)
	auth.POST("/reset-password", h.ResetPassword)
	auth.GET("/me", authMiddleware, h.Me) // Middleware auth required in main.go
}
