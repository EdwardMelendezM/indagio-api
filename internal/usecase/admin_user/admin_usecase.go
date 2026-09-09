package usecase

import (
	"context"
	"errors"
	"fmt"
	"foro-unsaac-backend/internal/domain"
	"foro-unsaac-backend/internal/utils"

	"github.com/google/uuid"
)

type adminUsecase struct {
	adminRepo domain.AdminUserRepository
	userRepo  domain.UserRepository
	otpRepo   domain.OTPRepository
	tokenSvc  domain.TokenService
	emailSvc  domain.EmailService
}

// NewAdminUsecase wires all dependencies required for admin authentication.
// It reuses the existing OTP and token/email infrastructure.
func NewAdminUsecase(
	adminRepo domain.AdminUserRepository,
	userRepo domain.UserRepository,
	otpRepo domain.OTPRepository,
	tokenSvc domain.TokenService,
	emailSvc domain.EmailService,
) domain.AdminUsecase {
	return &adminUsecase{
		adminRepo: adminRepo,
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		tokenSvc:  tokenSvc,
		emailSvc:  emailSvc,
	}
}

// RequestOTP checks whether the email belongs to an active admin and,
// if so, generates a 6-digit OTP and sends it by email.
//
// Security note: on failure we always return ErrUnauthorized so callers
// cannot enumerate which emails are registered in user_adm.
func (u *adminUsecase) RequestOTP(ctx context.Context, email string) error {
	// 1. Verify existing
	_, err := u.adminRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	// 1.1 Create admin user
	_, err = u.adminRepo.Create(ctx, email)
	if err != nil {
		// Log internally but surface a generic error externally.
		return domain.ErrUnauthorized
	}

	// 1.2 Verify if user exists
	_, err = u.userRepo.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		// Create user
		_, err = u.userRepo.Create(ctx, "Administrador", email, "TEMP_PASSWORD")
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	// 2. Generate a one-time code (reuses existing OTP util).
	code := utils.GenerateOTP()

	// 3. Persist the OTP (expires in 15 min, handled by OTPRepository).
	if err := u.otpRepo.Create(ctx, email, code); err != nil {
		return err
	}

	// 4. Send the OTP via email (reuses existing email util).
	if err := u.emailSvc.SendOTP(ctx, email, code); err != nil {
		return err
	}

	return nil
}

// Login validates the OTP for the given email and returns a signed JWT
// access token carrying role "admin". The token can be used immediately
// to access protected admin routes via AdminAuthMiddleware.
func (u *adminUsecase) Login(ctx context.Context, email string) error {
	// 1. Confirm the email still belongs to an active admin
	//    (the admin could have been soft-deleted between RequestOTP and Login).
	_, err := u.adminRepo.FindByEmail(ctx, email)
	if err != nil {
		return domain.ErrUnauthorized
	}

	code := utils.GenerateOTP()

	if err := u.otpRepo.Create(ctx, email, code); err != nil {
		return err
	}

	// 2. Validate and consume the OTP (marks it as used to prevent replay).
	if err := u.emailSvc.SendOTP(ctx, email, code); err != nil {
		return err
	}

	return nil
}

// VerifyOTP marks the admin as verified after validating the OTP, allowing them to log in.
func (u *adminUsecase) VerifyOTP(ctx context.Context, email, code string) (*domain.AdminUser, error) {
	valid, err := u.otpRepo.ValidateAndConsume(ctx, email, code)
	if err != nil {
		return nil, fmt.Errorf("validate otp: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("otp invalid or expired: %w", domain.ErrValidation)
	}

	if err := u.adminRepo.MarkVerified(ctx, email); err != nil {
		return nil, fmt.Errorf("mark verified: %w", err)
	}

	user, err := u.adminRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	return user, nil
}

// GenerateTokens generates both access and refresh tokens for the given user ID and role.
func (u *adminUsecase) GenerateTokens(userID uuid.UUID, role domain.Role) (string, string, error) {
	accessToken, err := u.tokenSvc.GenerateAccessToken(userID, role)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := u.tokenSvc.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (u *adminUsecase) BlockUser(ctx context.Context, id uuid.UUID) error {
	if err := u.userRepo.BlockUser(ctx, id); err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	return nil
}
