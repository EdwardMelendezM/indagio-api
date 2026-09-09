package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"foro-unsaac-backend/internal/domain"
	"foro-unsaac-backend/internal/domain/mocks"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runMiddleware wires OptionalAuthMiddleware ahead of a noop handler
// and executes a single request through the chain. Returns the
// response recorder and the captured userID/role from the context.
func runMiddleware(t *testing.T, tokenSvc domain.TokenService, header string) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.New()
	r.Use(OptionalAuthMiddleware(tokenSvc))
	r.GET("/probe", func(c *gin.Context) {
		// Echo back what landed in the context. The handler itself
		// is the test target — the middleware decides whether to
		// populate these keys.
		if uid, ok := c.Get("userID"); ok {
			c.JSON(http.StatusOK, gin.H{"userID": uid})
			return
		}
		c.JSON(http.StatusOK, gin.H{"userID": nil})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOptionalAuth_NoHeader_ContinuesAnonymously(t *testing.T) {
	tokenSvc := &mocks.TokenService{}
	// No EXPECT — without a header, the middleware must not even
	// touch the token service.

	w := runMiddleware(t, tokenSvc, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":null`)
	tokenSvc.AssertExpectations(t)
}

func TestOptionalAuth_NonBearerHeader_ContinuesAnonymously(t *testing.T) {
	tokenSvc := &mocks.TokenService{}
	// "Basic ..." is not a Bearer token; we skip without calling.

	w := runMiddleware(t, tokenSvc, "Basic dXNlcjpwYXNz")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"userID":null`)
	tokenSvc.AssertExpectations(t)
}

func TestOptionalAuth_InvalidToken_ContinuesAnonymously(t *testing.T) {
	tokenSvc := &mocks.TokenService{}
	tokenSvc.On("ValidateAccessToken", "garbage").Return(nil, errors.New("signature invalid")).Once()

	w := runMiddleware(t, tokenSvc, "Bearer garbage")
	assert.Equal(t, http.StatusOK, w.Code, "invalid token on optional endpoint must not 401")
	assert.Contains(t, w.Body.String(), `"userID":null`)
	tokenSvc.AssertExpectations(t)
}

func TestOptionalAuth_ValidToken_InjectsUserIDAndRole(t *testing.T) {
	userID := uuid.New()

	tokenSvc := &mocks.TokenService{}
	claims := &domain.TokenClaims{
		UserID: userID,
		Role:   domain.RoleStudent,
	}
	tokenSvc.On("ValidateAccessToken", "valid-token").Return(claims, nil).Once()

	r := gin.New()
	r.Use(OptionalAuthMiddleware(tokenSvc))
	r.GET("/probe", func(c *gin.Context) {
		uid, hasUID := c.Get("userID")
		role, hasRole := c.Get("userRole")
		c.JSON(http.StatusOK, gin.H{
			"hasUID":  hasUID,
			"hasRole": hasRole,
			"uid":     uid,
			"role":    role,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"hasUID":true`)
	assert.Contains(t, w.Body.String(), `"hasRole":true`)
	assert.Contains(t, w.Body.String(), `"uid":"`+userID.String()+`"`)
	assert.Contains(t, w.Body.String(), `"role":"estudiante"`)
	tokenSvc.AssertExpectations(t)
}

// Defensive — assert the middleware does not abort on a missing
// context key, and that downstream handlers can still read what the
// middleware wrote. This is the contract that handler-side
// `c.MustGet("userID")` calls rely on for the auth-required path,
// and that the avatar_border handler will rely on for the
// optional path (c.Get with the ok-check).
func TestOptionalAuth_DoesNotCallNextOnPanic(t *testing.T) {
	tokenSvc := &mocks.TokenService{}
	tokenSvc.On("ValidateAccessToken", mock.Anything).Return(nil, errors.New("boom")).Once()

	// Just ensure the middleware survives an upstream error and
	// still lets the request through.
	w := runMiddleware(t, tokenSvc, "Bearer whatever")
	assert.Equal(t, http.StatusOK, w.Code)
}
