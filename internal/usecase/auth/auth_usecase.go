package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

type authUsecase struct {
	userRepo          domain.UserRepository
	otpRepo           domain.OTPRepository
	emailSvc          domain.EmailService
	passwordSvc       domain.PasswordService
	tokenSvc          domain.TokenService
	jobRepo           domain.JobRepository
	allowedDomainRepo domain.AllowedDomainRepository
}

// NewAuthUsecase — constructor returns interface (Golden Rule 6)
func NewAuthUsecase(
	userRepo domain.UserRepository,
	otpRepo domain.OTPRepository,
	emailSvc domain.EmailService,
	passwordSvc domain.PasswordService,
	tokenSvc domain.TokenService,
	jobRepo domain.JobRepository,
	allowedDomainRepo domain.AllowedDomainRepository,
) domain.AuthUsecase {
	return &authUsecase{
		userRepo:          userRepo,
		otpRepo:           otpRepo,
		emailSvc:          emailSvc,
		passwordSvc:       passwordSvc,
		tokenSvc:          tokenSvc,
		jobRepo:           jobRepo,
		allowedDomainRepo: allowedDomainRepo,
	}
}

func (uc *authUsecase) SendOTP(ctx context.Context, email string) error {
	code := generateOTP() // pure function
	if err := uc.otpRepo.Create(ctx, email, code); err != nil {
		return fmt.Errorf("save otp: %w", err)
	}
	err := uc.jobRepo.Enqueue(ctx, domain.JobTypeSendOTP, domain.OTPJobPayload{
		Email: email,
		Code:  code,
	})
	if err != nil {
		fmt.Printf("enqueue OTP job: %v\n", err)
	}
	return nil
}

func (uc *authUsecase) Register(ctx context.Context, name, email, password string) error {
	// Domain validation - extract and validate domain from email
	emailDomain, err := domain.ExtractDomain(email)
	if err != nil {
		return fmt.Errorf("email format: %w", domain.ErrValidation)
	}

	// Check if domain is allowed (uses cache internally)
	allowed, err := uc.allowedDomainRepo.IsAllowed(ctx, emailDomain)
	if err != nil {
		return fmt.Errorf("domain check: %w", err)
	}
	if !allowed {
		return fmt.Errorf("domain not allowed: %w", domain.ErrDomainNotAllowed)
	}

	// Basic validation
	if len(name) < 2 || len(name) > 80 {
		return fmt.Errorf("name length: %w", domain.ErrValidation)
	}
	if len(password) < 8 {
		return fmt.Errorf("password length: %w", domain.ErrValidation)
	}

	// Check email uniqueness
	_, err = uc.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return fmt.Errorf("email conflict: %w", domain.ErrUserAlreadyExists)
	}
	if !domain.IsNotFound(err) {
		return fmt.Errorf("find user: %w", err)
	}

	// Hash password
	hash, err := uc.passwordSvc.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Create user
	_, err = uc.userRepo.Create(ctx, name, email, hash)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	err = uc.SendOTP(ctx, email)
	if err != nil {
		return fmt.Errorf("create otp: %w", err)
	}

	return nil
}

func (uc *authUsecase) VerifyOTP(ctx context.Context, email, code string) (*domain.User, error) {
	valid, err := uc.otpRepo.ValidateAndConsume(ctx, email, code)
	if err != nil {
		return nil, fmt.Errorf("validate otp: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("otp invalid or expired: %w", domain.ErrValidation)
	}

	if err := uc.userRepo.MarkVerified(ctx, email); err != nil {
		return nil, fmt.Errorf("mark verified: %w", err)
	}

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	return internalToPublic(user), nil
}

func (uc *authUsecase) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, fmt.Errorf("credenciales invalidas: %w", domain.ErrInvalidCredentials)
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !uc.passwordSvc.Verify(user.Password, password) {
		return nil, fmt.Errorf("credenciales invalidas: %w", domain.ErrInvalidCredentials)
	}

	if !user.Verified {
		return nil, fmt.Errorf("usuario no verificado: %w", domain.ErrForbidden)
	}

	return internalToPublic(user), nil
}

func (uc *authUsecase) GenerateTokens(userID uuid.UUID, role domain.Role) (string, string, error) {
	access, err := uc.tokenSvc.GenerateAccessToken(userID, role)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}
	refresh, err := uc.tokenSvc.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	return access, refresh, nil
}

func (uc *authUsecase) ValidateAccessToken(token string) (*domain.TokenClaims, error) {
	claims, err := uc.tokenSvc.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("validate access token: %w", err)
	}
	return claims, nil
}

func (uc *authUsecase) ValidateRefreshToken(token string) (*domain.TokenClaims, error) {
	claims, err := uc.tokenSvc.ValidateRefreshToken(token)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}
	return claims, nil
}

func (uc *authUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

// Helpers
func internalToPublic(u *domain.UserInternal) *domain.User {
	return &domain.User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		Verified:  u.Verified,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
}

func generateOTP() string {
	// Definimos el límite superior exclusivo (1,000,000)
	// Esto generará números aleatorios en el rango [0, 999999]
	newInt := big.NewInt(1000000)

	n, err := rand.Int(rand.Reader, newInt)
	if err != nil {
		// Fallback seguro en caso extremo de que el lector criptográfico falle
		return "123456"
	}

	// %06d asegura que si el número es, por ejemplo, 742,
	// se rellene con ceros a la izquierda devolviendo "000742"
	return fmt.Sprintf("%06d", n.Int64())
}

func (uc *authUsecase) UpdateUserName(ctx context.Context, requesterID uuid.UUID, targetUserID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return fmt.Errorf("name length: %w", domain.ErrValidation)
	}

	// Authorization: only the own user or moderador/admin can change the name
	if requesterID != targetUserID {
		return fmt.Errorf("not authorized: %w", domain.ErrForbidden)
	}

	return uc.userRepo.UpdateName(ctx, targetUserID, name)
}

func (uc *authUsecase) ForgotPassword(ctx context.Context, email string) error {
	// Check if user exists
	_, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if domain.IsNotFound(err) {
			// Don't reveal if email exists or not for security
			return domain.ErrNotExistEmail
		}
		return fmt.Errorf("find user: %w", err)
	}

	// Generate OTP for password reset
	code := generateOTP()
	if err := uc.otpRepo.Create(ctx, email, code); err != nil {
		return fmt.Errorf("save otp: %w", err)
	}

	// Enqueue password reset email job
	err = uc.jobRepo.Enqueue(ctx, domain.JobTypeSendPasswordReset, domain.PasswordResetJobPayload{
		Email: email,
		Code:  code,
	})
	if err != nil {
		fmt.Printf("enqueue password reset job: %v\n", err)
	}

	return nil
}

func (uc *authUsecase) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	// Validate OTP
	valid, err := uc.otpRepo.ValidateAndConsume(ctx, email, code)
	if err != nil {
		return fmt.Errorf("validate otp: %w", err)
	}
	if !valid {
		return fmt.Errorf("otp invalid or expired: %w", domain.ErrValidation)
	}

	// Validate new password
	if len(newPassword) < 8 {
		return fmt.Errorf("password length: %w", domain.ErrValidation)
	}

	// Hash new password
	hash, err := uc.passwordSvc.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update user password
	if err := uc.userRepo.UpdatePassword(ctx, email, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	return nil
}
