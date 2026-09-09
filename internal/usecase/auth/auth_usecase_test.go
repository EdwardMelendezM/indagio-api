package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"indagio-api/internal/domain"
	"indagio-api/internal/domain/mocks"
)

// TestAuthUsecase_Register tests user registration flow
func TestAuthUsecase_Register(t *testing.T) {
	t.Run("When register user successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		name := "John Doe"
		email := "john@unsaac.edu.pe"
		password := "ValidPassword123!"
		hash := "$2a$12$hashedpassword"
		userID := uuid.New()

		// Setup expectations

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(nil, domain.ErrNotFound)

		passwordSvc.On("Hash", password).
			Return(hash, nil)

		user := &domain.User{
			ID:       userID,
			Name:     name,
			Email:    email,
			Role:     domain.RoleStudent,
			Verified: false,
		}

		userRepo.On("Create", mock.Anything, name, email, hash).
			Return(user, nil)

		otpRepo.On("Create", mock.Anything, email, mock.AnythingOfType("string")).
			Return(nil)

		// Mock email service (called async, so we need to expect it)
		emailSvc.On("SendOTP", mock.Anything, email, mock.AnythingOfType("string")).
			Return(nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		// Create usecase and test
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.Register(context.Background(), name, email, password)

		assert.NoError(t, err)
		userRepo.AssertCalled(t, "FindByEmail", mock.Anything, email)
		userRepo.AssertCalled(t, "Create", mock.Anything, name, email, hash)
		otpRepo.AssertCalled(t, "Create", mock.Anything, email, mock.AnythingOfType("string"))
	})

	t.Run("When email already exists", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "existing@unsaac.edu.pe"
		existingUser := &domain.UserInternal{
			ID:    uuid.New(),
			Email: email,
		}

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(existingUser, nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.Register(context.Background(), "John", email, "Password123!")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrUserAlreadyExists))
	})

	t.Run("When password too short", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.Register(context.Background(), "John", "john@unsaac.edu.pe", "short")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})

	t.Run("When password hashing fails", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		password := "ValidPassword123!"

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(nil, domain.ErrNotFound)

		passwordSvc.On("Hash", password).
			Return("", errors.New("hash error"))

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.Register(context.Background(), "John", email, password)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hash password")
	})
}

// TestAuthUsecase_VerifyOTP tests OTP verification flow
func TestAuthUsecase_VerifyOTP(t *testing.T) {
	t.Run("When verify OTP successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		code := "123456"
		userID := uuid.New()

		otpRepo.On("ValidateAndConsume", mock.Anything, email, code).
			Return(true, nil)

		userRepo.On("MarkVerified", mock.Anything, email).
			Return(nil)

		user := &domain.UserInternal{
			ID:       userID,
			Name:     "John",
			Email:    email,
			Verified: true,
		}

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(user, nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.VerifyOTP(context.Background(), email, code)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, email, result.Email)
		assert.Equal(t, true, result.Verified)
	})

	t.Run("When OTP invalid or expired", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		code := "invalid"

		otpRepo.On("ValidateAndConsume", mock.Anything, email, code).
			Return(false, nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.VerifyOTP(context.Background(), email, code)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})
}

// TestAuthUsecase_Login tests login flow
func TestAuthUsecase_Login(t *testing.T) {
	t.Run("When login successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		password := "CorrectPassword123!"
		hash := "$2a$12$hashedpassword"
		userID := uuid.New()

		userInternal := &domain.UserInternal{
			ID:       userID,
			Email:    email,
			Password: hash,
			Verified: true,
		}

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(userInternal, nil)

		passwordSvc.On("Verify", hash, password).
			Return(true)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.Login(context.Background(), email, password)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, email, result.Email)
	})

	t.Run("When user not found", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "nonexistent@unsaac.edu.pe"

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(nil, domain.ErrNotFound)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.Login(context.Background(), email, "password")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrInvalidCredentials))
	})

	t.Run("When password incorrect", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		hash := "$2a$12$hashedpassword"

		userInternal := &domain.UserInternal{
			ID:       uuid.New(),
			Email:    email,
			Password: hash,
			Verified: true,
		}

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(userInternal, nil)

		passwordSvc.On("Verify", hash, "WrongPassword").
			Return(false)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.Login(context.Background(), email, "WrongPassword")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrInvalidCredentials))
	})

	t.Run("When user not verified", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		email := "john@unsaac.edu.pe"
		hash := "$2a$12$hashedpassword"

		userInternal := &domain.UserInternal{
			ID:       uuid.New(),
			Email:    email,
			Password: hash,
			Verified: false, // Not verified
		}

		userRepo.On("FindByEmail", mock.Anything, email).
			Return(userInternal, nil)

		passwordSvc.On("Verify", hash, "CorrectPassword123!").
			Return(true)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.Login(context.Background(), email, "CorrectPassword123!")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrForbidden))
	})
}

// TestAuthUsecase_GenerateTokens tests token generation
func TestAuthUsecase_GenerateTokens(t *testing.T) {
	t.Run("When generate tokens successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()
		role := domain.RoleStudent
		accessToken := "access-token-jwt"
		refreshToken := "refresh-token-jwt"

		tokenSvc.On("GenerateAccessToken", userID, role).
			Return(accessToken, nil)

		tokenSvc.On("GenerateRefreshToken", userID).
			Return(refreshToken, nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		access, refresh, err := uc.GenerateTokens(userID, role)

		assert.NoError(t, err)
		assert.Equal(t, accessToken, access)
		assert.Equal(t, refreshToken, refresh)
	})

	t.Run("When access token generation fails", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()
		role := domain.RoleStudent

		tokenSvc.On("GenerateAccessToken", userID, role).
			Return("", errors.New("token error"))

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		access, refresh, err := uc.GenerateTokens(userID, role)

		assert.Error(t, err)
		assert.Empty(t, access)
		assert.Empty(t, refresh)
	})
}

// TestAuthUsecase_UpdateUserName tests updating user name
func TestAuthUsecase_UpdateUserName(t *testing.T) {
	t.Run("When user updates own name successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()
		newName := "Updated Name"

		userRepo.On("UpdateName", mock.Anything, userID, newName).
			Return(nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.UpdateUserName(context.Background(), userID, userID, newName)

		assert.NoError(t, err)
		userRepo.AssertCalled(t, "UpdateName", mock.Anything, userID, newName)
	})

	t.Run("When different user tries to update another's name", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		requesterID := uuid.New()
		targetUserID := uuid.New()
		newName := "Moderator Changed"

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.UpdateUserName(context.Background(), requesterID, targetUserID, newName)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrForbidden))
	})

	t.Run("When name too short", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		requesterID := uuid.New()
		targetUserID := uuid.New()

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.UpdateUserName(context.Background(), requesterID, targetUserID, "x")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrValidation))
	})

	t.Run("When unauthorized user tries to update another user's name", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		requesterID := uuid.New()
		targetUserID := uuid.New() // different from requester

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.UpdateUserName(context.Background(), requesterID, targetUserID, "Some Name")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, domain.ErrForbidden))
	})

	t.Run("When repository returns error", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()
		newName := "Updated Name"

		userRepo.On("UpdateName", mock.Anything, userID, newName).
			Return(errors.New("db error"))

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		err := uc.UpdateUserName(context.Background(), userID, userID, newName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

// TestAuthUsecase_GetUserByID tests fetching user by ID
func TestAuthUsecase_GetUserByID(t *testing.T) {
	t.Run("When get user by ID successfully", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()
		user := &domain.User{
			ID:       userID,
			Name:     "John",
			Email:    "john@unsaac.edu.pe",
			Verified: true,
		}

		userRepo.On("FindByID", mock.Anything, userID).
			Return(user, nil)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.GetUserByID(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, userID, result.ID)
	})

	t.Run("When user not found by ID", func(t *testing.T) {
		userRepo := new(mocks.UserRepository)
		otpRepo := new(mocks.OTPRepository)
		emailSvc := new(mocks.EmailService)
		passwordSvc := new(mocks.PasswordService)
		tokenSvc := new(mocks.TokenService)
		jobRepo := new(mocks.JobRepository)

		userID := uuid.New()

		userRepo.On("FindByID", mock.Anything, userID).
			Return(nil, domain.ErrNotFound)

		jobRepo.On("Enqueue", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
		uc := NewAuthUsecase(userRepo, otpRepo, emailSvc, passwordSvc, tokenSvc, jobRepo)
		result, err := uc.GetUserByID(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
