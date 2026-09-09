package media

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"indagio-api/internal/domain"
	"indagio-api/internal/domain/mocks"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// buildAdminRouter wires RegisterBorderPresignRoute with a stub
// adminMiddleware that always injects a moderator userID + role.
// Real auth is exercised in the auth package's handler tests.
func buildAdminRouter(h *BorderPresignHandler) *gin.Engine {
	r := gin.New()
	admin := func(c *gin.Context) {
		// userID isn't required by the handler, but the auth
		// middleware would normally inject it. We skip the
		// injection to keep the test focused on the presign logic.
		c.Set("userRole", domain.RoleModerator)
		c.Next()
	}
	rg := r.Group("/api/admin")
	RegisterBorderPresignRoute(rg, h, admin)
	return r
}

func newOKStorage(t *testing.T) *mocks.StorageRepository {
	t.Helper()
	s := &mocks.StorageRepository{}
	s.On("GetPresignedUploadURL", mock.Anything, mock.Anything, mock.Anything).
		Return("https://r2.example.com/upload-url?X-Amz-…", nil).Once()
	return s
}

// ─────────────────────────────────────────────────────────────────────
// Happy path
// ─────────────────────────────────────────────────────────────────────

func TestBorderPresign_ValidRequest_ReturnsURL(t *testing.T) {
	storage := newOKStorage(t)
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 200000,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var got PresignedBorderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Contains(t, got.UploadURL, "https://r2.example.com/upload-url")
	assert.Contains(t, got.AssetURL, "https://cdn.example.com/borders/")
	assert.Contains(t, got.AssetURL, ".png")
	assert.Equal(t, int64(200000), got.MaxSizeBytes)
	storage.AssertExpectations(t)
}

func TestBorderPresign_WebP_Accepted(t *testing.T) {
	storage := newOKStorage(t)
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"content_type": "image/webp",
		"max_size_bytes": 100000,
		"extension": "webp"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got PresignedBorderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Contains(t, got.AssetURL, ".webp")
	storage.AssertExpectations(t)
}

// ─────────────────────────────────────────────────────────────────────
// Size clamping
// ─────────────────────────────────────────────────────────────────────

func TestBorderPresign_ClampsSizeToServerCap(t *testing.T) {
	storage := newOKStorage(t)
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	// 10 MB requested — server caps to 512 KB.
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 10485760,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got PresignedBorderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, DefaultBorderMaxSizeBytes, got.MaxSizeBytes, "client request above the cap must be clamped")
	storage.AssertExpectations(t)
}

// ─────────────────────────────────────────────────────────────────────
// Validation failures
// ─────────────────────────────────────────────────────────────────────

func TestBorderPresign_InvalidContentType_Returns422(t *testing.T) {
	storage := &mocks.StorageRepository{} // no expectations — handler must short-circuit
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	// JPEG has no alpha — borders need transparency.
	body := bytes.NewBufferString(`{
		"content_type": "image/jpeg",
		"max_size_bytes": 200000,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	storage.AssertExpectations(t)
}

func TestBorderPresign_InvalidExtension_Returns422(t *testing.T) {
	storage := &mocks.StorageRepository{}
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	// GIF excluded in phase 1 (sticker-only).
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 200000,
		"extension": "gif"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	storage.AssertExpectations(t)
}

func TestBorderPresign_MissingContentType_Returns422(t *testing.T) {
	storage := &mocks.StorageRepository{}
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"max_size_bytes": 200000,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	storage.AssertExpectations(t)
}

func TestBorderPresign_MissingExtension_Returns422(t *testing.T) {
	storage := &mocks.StorageRepository{}
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 200000
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	storage.AssertExpectations(t)
}

func TestBorderPresign_MalformedJSON_Returns422(t *testing.T) {
	storage := &mocks.StorageRepository{}
	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	storage.AssertExpectations(t)
}

// ─────────────────────────────────────────────────────────────────────
// Storage backend failure
// ─────────────────────────────────────────────────────────────────────

func TestBorderPresign_StorageError_Returns500(t *testing.T) {
	storage := &mocks.StorageRepository{}
	storage.On("GetPresignedUploadURL", mock.Anything, mock.Anything, mock.Anything).
		Return("", assert.AnError).Once()

	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 200000,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	storage.AssertExpectations(t)
}

// ─────────────────────────────────────────────────────────────────────
// Key prefix invariant
// ─────────────────────────────────────────────────────────────────────

func TestBorderPresign_AssetURLUsesBordersPrefix(t *testing.T) {
	storage := &mocks.StorageRepository{}
	storage.On("GetPresignedUploadURL", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return("https://r2.example.com/upload-url", nil).Once()

	h := NewBorderPresignHandler(BorderPresignDeps{
		Storage:      storage,
		Logger:       silentLogger(),
		PublicDomain: "https://cdn.example.com",
	})
	r := buildAdminRouter(h)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{
		"content_type": "image/png",
		"max_size_bytes": 200000,
		"extension": "png"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/presign-border", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got PresignedBorderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	// The asset_url must start with `<publicDomain>/borders/` and
	// end with the requested extension. No per-user prefix (catalog-
	// level), no pending/ subfolder (admin commits immediately).
	assert.Contains(t, got.AssetURL, "https://cdn.example.com/borders/")
	assert.Contains(t, got.AssetURL, ".png")
	assert.NotContains(t, got.AssetURL, "/pending/")
	assert.NotContains(t, got.AssetURL, "/users/")
	storage.AssertExpectations(t)
}
