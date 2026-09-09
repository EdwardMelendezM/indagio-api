package user

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"foro-unsaac-backend/internal/domain"
	"foro-unsaac-backend/internal/domain/mocks"
)

func setupUserRouter(handler *UserHandler, authMiddleware func(*gin.Context)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterUserRoutes(r.Group("/api"), handler, authMiddleware)
	return r
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestUserHandler_UpdateName(t *testing.T) {
	t.Run("When update name successfully", func(t *testing.T) {
		mockAuthUsecase := new(mocks.AuthUsecase)
		// userUC is not exercised by UpdateName — pass nil.
		handler := NewUserHandler(mockAuthUsecase, nil, newLogger())

		requesterID := uuid.New()
		targetUserID := uuid.New()
		newName := "Updated Name"

		mockAuthUsecase.On("UpdateUserName", mock.Anything, requesterID, targetUserID, newName).
			Return(nil)

		router := setupUserRouter(handler, func(c *gin.Context) {
			c.Set("userID", requesterID)
			c.Set("userRole", "estudiante")
			c.Next()
		})

		body := map[string]string{"name": newName}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPatch, "/api/users/"+targetUserID.String()+"/name", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "name updated", resp["message"])
		mockAuthUsecase.AssertExpectations(t)
	})

	t.Run("When update name with invalid user id", func(t *testing.T) {
		mockAuthUsecase := new(mocks.AuthUsecase)
		// userUC is not exercised by UpdateName — pass nil.
		handler := NewUserHandler(mockAuthUsecase, nil, newLogger())

		router := setupUserRouter(handler, func(c *gin.Context) {
			c.Set("userID", uuid.New())
			c.Set("userRole", "estudiante")
			c.Next()
		})

		body := map[string]string{"name": "Valid Name"}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPatch, "/api/users/invalid-uuid/name", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "invalid user id", resp["error"])
	})

	t.Run("When update name with invalid request body", func(t *testing.T) {
		mockAuthUsecase := new(mocks.AuthUsecase)
		// userUC is not exercised by UpdateName — pass nil.
		handler := NewUserHandler(mockAuthUsecase, nil, newLogger())

		requesterID := uuid.New()
		targetUserID := uuid.New()

		router := setupUserRouter(handler, func(c *gin.Context) {
			c.Set("userID", requesterID)
			c.Set("userRole", "estudiante")
			c.Next()
		})

		req := httptest.NewRequest(http.MethodPatch, "/api/users/"+targetUserID.String()+"/name", bytes.NewBufferString(`{bad json`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("When update name with usecase error", func(t *testing.T) {
		mockAuthUsecase := new(mocks.AuthUsecase)
		// userUC is not exercised by UpdateName — pass nil.
		handler := NewUserHandler(mockAuthUsecase, nil, newLogger())

		requesterID := uuid.New()
		targetUserID := uuid.New()
		newName := "Valid Name"

		mockAuthUsecase.On("UpdateUserName", mock.Anything, requesterID, targetUserID, newName).
			Return(assert.AnError)

		router := setupUserRouter(handler, func(c *gin.Context) {
			c.Set("userID", requesterID)
			c.Set("userRole", "estudiante")
			c.Next()
		})

		body := map[string]string{"name": newName}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPatch, "/api/users/"+targetUserID.String()+"/name", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockAuthUsecase.AssertExpectations(t)
	})

	t.Run("When update name with forbidden error", func(t *testing.T) {
		mockAuthUsecase := new(mocks.AuthUsecase)
		// userUC is not exercised by UpdateName — pass nil.
		handler := NewUserHandler(mockAuthUsecase, nil, newLogger())

		requesterID := uuid.New()
		targetUserID := uuid.New()

		mockAuthUsecase.On("UpdateUserName", mock.Anything, requesterID, targetUserID, "New Name").
			Return(domain.ErrForbidden)

		router := setupUserRouter(handler, func(c *gin.Context) {
			c.Set("userID", requesterID)
			c.Set("userRole", "estudiante")
			c.Next()
		})

		body := map[string]string{"name": "New Name"}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPatch, "/api/users/"+targetUserID.String()+"/name", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockAuthUsecase.AssertExpectations(t)
	})
}
