package auth

import (
	"bytes"
	"encoding/json"
	"errors"

	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"foro-unsaac-backend/internal/domain"
	"foro-unsaac-backend/internal/domain/mocks"
)

// ---------------------------------------------------------------------------
// Helpers de test
// ---------------------------------------------------------------------------

func setupRouter(handler *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authMiddleware := func(c *gin.Context) { c.Next() } // stub; se sobreescribe en Me tests
	RegisterAuthRoutes(r.Group("/api"), handler, authMiddleware)
	return r
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func sampleUser() *domain.User {
	avatarURL := "https://example.com/avatar.png"
	return &domain.User{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "Test User",
		Email:     "test@unsaac.edu.pe",
		Role:      domain.RoleStudent,
		AvatarURL: &avatarURL,
		CreatedAt: time.Now(),
	}
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return bytes.NewReader(b)
}

// ---------------------------------------------------------------------------
// POST /api/auth/register
// ---------------------------------------------------------------------------

func TestAuthHandler_Register_Success(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("Register", mock.Anything, "Test User", "test@unsaac.edu.pe", "Password123!").
		Return(nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(t, map[string]string{
		"name":     "Test User",
		"email":    "test@unsaac.edu.pe",
		"password": "Password123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "message")
	uc.AssertExpectations(t)
}

func TestAuthHandler_Register_InvalidBody(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	// Body malformado
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "Register")
}

func TestAuthHandler_Register_MissingFields(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	// Falta password
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(t, map[string]string{
		"name":  "Test User",
		"email": "test@unsaac.edu.pe",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "Register")
}

func TestAuthHandler_Register_UsecaseError(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("Register", mock.Anything, "Test User", "dup@unsaac.edu.pe", "Password123!").
		Return(domain.ErrConflict)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(t, map[string]string{
		"name":     "Test User",
		"email":    "dup@unsaac.edu.pe",
		"password": "Password123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	uc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// POST /api/auth/verify-otp
// ---------------------------------------------------------------------------

func TestAuthHandler_VerifyOTP_Success(t *testing.T) {
	user := sampleUser()
	uc := new(mocks.AuthUsecase)
	uc.On("VerifyOTP", mock.Anything, "test@unsaac.edu.pe", "123456").Return(user, nil)
	uc.On("GenerateTokens", user.ID, user.Role).Return("access_tok", "refresh_tok", nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-otp", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
		"code":  "123456",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AuthResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access_tok", resp.AccessToken)
	assert.Equal(t, "refresh_tok", resp.RefreshToken)
	assert.Equal(t, user.Email, resp.User.Email)
	uc.AssertExpectations(t)
}

func TestAuthHandler_VerifyOTP_InvalidOTP(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("VerifyOTP", mock.Anything, "test@unsaac.edu.pe", "000000").
		Return(nil, domain.ErrUnauthorized)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-otp", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
		"code":  "000000",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

func TestAuthHandler_VerifyOTP_InvalidBody(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	// code tiene que tener exactamente 6 chars; aquí mandamos 3
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-otp", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
		"code":  "123",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "VerifyOTP")
}

func TestAuthHandler_VerifyOTP_TokenGenerationError(t *testing.T) {
	user := sampleUser()
	uc := new(mocks.AuthUsecase)
	uc.On("VerifyOTP", mock.Anything, "test@unsaac.edu.pe", "123456").Return(user, nil)
	uc.On("GenerateTokens", user.ID, user.Role).Return("", "", errors.New("jwt signing error"))

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-otp", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
		"code":  "123456",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	uc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// POST /api/auth/login
// ---------------------------------------------------------------------------

func TestAuthHandler_Login_Success(t *testing.T) {
	user := sampleUser()
	uc := new(mocks.AuthUsecase)
	uc.On("Login", mock.Anything, "test@unsaac.edu.pe", "Password123!").Return(user, nil)
	uc.On("GenerateTokens", user.ID, user.Role).Return("access_tok", "refresh_tok", nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(t, map[string]string{
		"email":    "test@unsaac.edu.pe",
		"password": "Password123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AuthResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access_tok", resp.AccessToken)
	assert.Equal(t, user.Name, resp.User.Name)

	// Verifica que se setea la cookie refresh_token
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			assert.Equal(t, "refresh_tok", c.Value)
		}
	}
	assert.True(t, found, "expected refresh_token cookie")
	uc.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("Login", mock.Anything, "test@unsaac.edu.pe", "wrongpass").
		Return(nil, domain.ErrUnauthorized)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(t, map[string]string{
		"email":    "test@unsaac.edu.pe",
		"password": "wrongpass",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

func TestAuthHandler_Login_UserNotFound(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("Login", mock.Anything, "ghost@unsaac.edu.pe", "Password123!").
		Return(nil, domain.ErrNotFound)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(t, map[string]string{
		"email":    "ghost@unsaac.edu.pe",
		"password": "Password123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	uc.AssertExpectations(t)
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	// Falta password
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "Login")
}

// ---------------------------------------------------------------------------
// POST /api/auth/refresh
// ---------------------------------------------------------------------------

func TestAuthHandler_Refresh_Success(t *testing.T) {
	user := sampleUser()
	claims := &domain.TokenClaims{UserID: user.ID}

	uc := new(mocks.AuthUsecase)
	uc.On("ValidateRefreshToken", "valid_refresh").Return(claims, nil)
	uc.On("GetUserByID", mock.Anything, user.ID).Return(user, nil)
	uc.On("GenerateTokens", user.ID, user.Role).Return("new_access", "new_refresh", nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid_refresh"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "new_access", resp["access_token"])
	uc.AssertExpectations(t)
}

func TestAuthHandler_Refresh_MissingCookie(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	// Sin cookie

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertNotCalled(t, "ValidateRefreshToken")
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("ValidateRefreshToken", "bad_token").Return(nil, domain.ErrUnauthorized)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "bad_token"})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// POST /api/auth/logout
// ---------------------------------------------------------------------------

func TestAuthHandler_Logout_Success(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "logged out", resp["message"])

	// La cookie debe expirar (MaxAge negativo)
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			assert.Less(t, c.MaxAge, 0)
		}
	}
	assert.True(t, found, "expected cleared refresh_token cookie")
}

// ---------------------------------------------------------------------------
// GET /api/auth/me
// ---------------------------------------------------------------------------

func TestAuthHandler_Me_Success(t *testing.T) {
	user := sampleUser()
	uc := new(mocks.AuthUsecase)
	uc.On("GetUserByID", mock.Anything, user.ID).Return(user, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewAuthHandler(uc, newLogger())

	// Middleware que inyecta userID en el contexto
	authMW := func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	}
	RegisterAuthRoutes(r.Group("/api"), handler, authMW)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp UserResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, user.Email, resp.Email)
	assert.Equal(t, user.Name, resp.Name)
	uc.AssertExpectations(t)
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	uc := new(mocks.AuthUsecase)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewAuthHandler(uc, newLogger())

	// Middleware que NO inyecta userID (simula token ausente)
	authMW := func(c *gin.Context) {
		// No set "userID" → handler devuelve 401
		c.Next()
	}
	RegisterAuthRoutes(r.Group("/api"), handler, authMW)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	uc.AssertNotCalled(t, "GetUserByID")
}

func TestAuthHandler_Me_UserNotFound(t *testing.T) {
	user := sampleUser()
	uc := new(mocks.AuthUsecase)
	uc.On("GetUserByID", mock.Anything, user.ID).Return(nil, domain.ErrNotFound)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewAuthHandler(uc, newLogger())

	authMW := func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	}
	RegisterAuthRoutes(r.Group("/api"), handler, authMW)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	uc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// POST /api/auth/forgot-password
// ---------------------------------------------------------------------------

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("ForgotPassword", mock.Anything, "test@unsaac.edu.pe").Return(nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "message")
	uc.AssertExpectations(t)
}

func TestAuthHandler_ForgotPassword_InvalidEmail(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", jsonBody(t, map[string]string{
		"email": "invalid-email",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "ForgotPassword")
}

func TestAuthHandler_ForgotPassword_MissingEmail(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", jsonBody(t, map[string]string{}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "ForgotPassword")
}

// ---------------------------------------------------------------------------
// POST /api/auth/reset-password
// ---------------------------------------------------------------------------

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("ResetPassword", mock.Anything, "test@unsaac.edu.pe", "123456", "NewPassword123!").
		Return(nil)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", jsonBody(t, map[string]string{
		"email":        "test@unsaac.edu.pe",
		"code":         "123456",
		"new_password": "NewPassword123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "message")
	uc.AssertExpectations(t)
}

func TestAuthHandler_ResetPassword_InvalidCode(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	uc.On("ResetPassword", mock.Anything, "test@unsaac.edu.pe", "000000", "NewPassword123!").
		Return(domain.ErrValidation)

	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", jsonBody(t, map[string]string{
		"email":        "test@unsaac.edu.pe",
		"code":         "000000",
		"new_password": "NewPassword123!",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertExpectations(t)
}

func TestAuthHandler_ResetPassword_ShortPassword(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", jsonBody(t, map[string]string{
		"email":        "test@unsaac.edu.pe",
		"code":         "123456",
		"new_password": "short",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "ResetPassword")
}

func TestAuthHandler_ResetPassword_MissingFields(t *testing.T) {
	uc := new(mocks.AuthUsecase)
	r := setupRouter(NewAuthHandler(uc, newLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", jsonBody(t, map[string]string{
		"email": "test@unsaac.edu.pe",
	}))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	uc.AssertNotCalled(t, "ResetPassword")
}
