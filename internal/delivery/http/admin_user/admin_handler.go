package admin_user

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"foro-unsaac-backend/internal/delivery/http/utils"
	"foro-unsaac-backend/internal/domain"
)

// AdminHandler handles all HTTP requests related to admin authentication.
type AdminHandler struct {
	uc     domain.AdminUsecase
	logger *slog.Logger
}

// NewAdminHandler constructs an AdminHandler with its usecase dependency.
func NewAdminHandler(uc domain.AdminUsecase, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		uc:     uc,
		logger: logger,
	}
}

// ── Handlers ────────────────────────────────────────────────────────────────

// RequestOTP godoc
//
//	@Summary     Request admin OTP
//	@Description Sends a 6-digit one-time password to the admin's registered email.
//	@Tags        AdminAuth
//	@Accept      json
//	@Produce     json
//	@Param       body body AdminRequestOTPRequest true "Admin email"
//	@Success     200 {object} AdminLoginResponse "Message"
//	@Failure     400 {object} map[string]string
//	@Failure     401 {object} map[string]string
//	@Router      /admin/auth/request-otp [post]
func (h *AdminHandler) RequestOTP(c *gin.Context) {
	var req AdminRequestOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	if err := h.uc.RequestOTP(c.Request.Context(), req.Email); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusCreated, AdminLoginResponse{
		Message: "Verifica tu codigo en tu correo",
	})
}

// Login godoc
//
//	@Summary     Admin login
//	@Description Validates the OTP and returns a signed JWT access token with role "admin".
//	@Tags        AdminAuth
//	@Accept      json
//	@Produce     json
//	@Param       body body AdminLoginRequest true "Admin email"
//	@Success     200 {object} AdminLoginResponse
//	@Failure     400 {object} map[string]string
//	@Failure     401 {object} map[string]string
//	@Router      /admin/auth/login [post]
func (h *AdminHandler) Login(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	err := h.uc.Login(c.Request.Context(), req.Email)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, AdminLoginResponse{
		Message: "Verifica tu codigo en tu correo",
	})
}

// Me godoc
//
//	@Summary     Current admin identity
//	@Description Returns the ID of the authenticated admin extracted from the JWT.
//	@Tags        AdminAuth
//	@Security    BearerAuth
//	@Produce     json
//	@Success     200 {object} map[string]string
//	@Failure     401 {object} map[string]string
//	@Router      /admin/me [get]
func (h *AdminHandler) Me(c *gin.Context) {
	// AdminAuthMiddleware guarantees these values are set.
	adminID, _ := c.Get("admin_id")
	adminRole, _ := c.Get("admin_role")

	c.JSON(http.StatusOK, gin.H{
		"admin_id": adminID,
		"role":     adminRole,
	})
}

// VerifyOTP godoc
// @Summary		Verify OTP and activate account
// @Description	Verify the 6-digit code sent to email to activate account
// @Tags		AdminAuth
// @Accept		json
// @Produce		json
// @Param		body body VerifyAdminOTPRequest true "Email and OTP code"
// @Success		200 {object} AuthAdminResponse "User authenticated with tokens"
// @Failure		400 {object} map[string]string "Invalid request"
// @Failure		422 {object} map[string]string "Invalid or expired OTP"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/admin/auth/verify-otp [post]
func (h *AdminHandler) VerifyOTP(c *gin.Context) {
	var req VerifyAdminOTPRequest
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

	access, refresh, err := h.uc.GenerateTokens(user.ID, domain.RoleModerator)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	// Set httpOnly cookie
	c.SetCookie("refresh_token", refresh, 7*24*3600, "/api/auth", "", false, true)

	c.JSON(http.StatusOK, AuthAdminResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		Admin: UserResponse{
			ID:    user.ID.String(),
			Email: user.Email,
		},
	})
}

// RegisterAdminRoutes wires the admin routes onto the provided router.
func RegisterAdminRoutes(rg *gin.RouterGroup, h *AdminHandler, adminAuthMiddleware gin.HandlerFunc) {
	adminGroup := rg.Group("/admin")

	// ── Public ──────────────────────────────────────────────────────────────
	auth := adminGroup.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/verify-otp", h.VerifyOTP)
		auth.POST("/request-otp", adminAuthMiddleware, h.RequestOTP)
	}

	// ── Protected (requires valid admin JWT) ─────────────────────────────────
	protected := adminGroup.Group("")
	{
		protected.GET("/me", adminAuthMiddleware, h.Me)
	}
}
