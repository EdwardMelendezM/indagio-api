package media

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foro-unsaac-backend/internal/delivery/http/utils"
	"foro-unsaac-backend/internal/domain"
)

const (
	// BorderPresignedURLTTL is the lifetime of the presigned PUT URL.
	// 15 minutes matches the sticker presign endpoint and the R2 default.
	BorderPresignedURLTTL = 15 * 60

	// DefaultBorderMaxSizeBytes is the server-side cap that the FE
	// may not exceed. 512 KB matches stickers; border assets are small
	// PNGs with a transparent hole in the middle.
	DefaultBorderMaxSizeBytes int64 = 512 * 1024
)

// PresignedBorderRequest — body for POST /api/admin/media/presign-border.
// Only PNG is allowed in phase 1: borders require alpha for the
// transparent hole, and JPEG can't carry it. WEBP supports alpha
// and we accept it as a smaller alternative. SVG is excluded; the
// FE doesn't have an SVG parser for the catalog yet.
type PresignedBorderRequest struct {
	ContentType  string `json:"content_type"  binding:"required,oneof=image/png image/webp"`
	MaxSizeBytes int64  `json:"max_size_bytes" binding:"required,min=1"`
	Extension    string `json:"extension"     binding:"required,oneof=png webp"`
}

// PresignedBorderResponse — server-issued presigned PUT URL plus the
// final asset URL. The admin hands the asset_url to the
// POST /api/admin/avatar-borders endpoint to commit the border.
type PresignedBorderResponse struct {
	UploadURL    string `json:"upload_url"`     // URL PUT firmada para R2
	AssetURL     string `json:"asset_url"`      // URL pública final; se persiste en avatar_borders.asset_url
	MaxSizeBytes int64  `json:"max_size_bytes"` // server-clamped cap
}

// BorderPresignDeps bundles what the border presign handler needs.
// Smaller than StickerPresignDeps because borders have no per-user
// quota and no worker — admin uploads once, commits immediately.
type BorderPresignDeps struct {
	Storage domain.StorageRepository
	Logger  *slog.Logger
	// PublicDomain is the CDN base used to build the public asset_url.
	// Pulled from the same config field the legacy MediaHandler uses.
	PublicDomain string
}

// BorderPresignHandler issues presigned URLs for border asset and
// thumbnail uploads. Admin-only; see RegisterBorderPresignRoute for
// the auth wiring.
type BorderPresignHandler struct {
	storage      domain.StorageRepository
	logger       *slog.Logger
	publicDomain string
}

func NewBorderPresignHandler(deps BorderPresignDeps) *BorderPresignHandler {
	return &BorderPresignHandler{
		storage:      deps.Storage,
		logger:       deps.Logger,
		publicDomain: deps.PublicDomain,
	}
}

// GetPresignedBorderURL — POST /api/admin/media/presign-border
//
// Returns a presigned PUT URL for the asset or thumbnail of a new
// avatar border. The key prefix is `borders/<uuid>.<ext>` — global,
// not per-user, because borders are catalog-level (admin-owned).
//
// The route is wired behind adminMiddleware; the handler itself does
// not re-check the role. Returns 422 on validation failure, 500 on
// storage backend failure.
//
// The caller is expected to PUT the file at upload_url, then hand
// asset_url to POST /api/admin/avatar-borders to commit the catalog
// row. Two presign calls are required per border (asset + thumbnail);
// each returns its own asset_url.
func (h *BorderPresignHandler) GetPresignedBorderURL(c *gin.Context) {
	var req PresignedBorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Clamp the requested size to the server cap so the FE can't
	// talk the server into issuing URLs for arbitrarily large files.
	effectiveMax := req.MaxSizeBytes
	if effectiveMax > DefaultBorderMaxSizeBytes {
		effectiveMax = DefaultBorderMaxSizeBytes
	}

	// No per-user prefix: borders are global. `borders/<uuid>.<ext>`
	// — no `pending/` subfolder because the admin commits the border
	// in the same request flow (no worker step).
	fileUUID := uuid.New().String()
	rawKey := fmt.Sprintf("borders/%s.%s", fileUUID, req.Extension)

	presignedURL, err := h.storage.GetPresignedUploadURL(c.Request.Context(), rawKey, req.ContentType)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	assetURL := fmt.Sprintf("%s/%s", h.publicDomain, rawKey)

	c.JSON(http.StatusOK, PresignedBorderResponse{
		UploadURL:    presignedURL,
		AssetURL:     assetURL,
		MaxSizeBytes: effectiveMax,
	})
}

// RegisterBorderPresignRoute mounts the border presign endpoint on
// the admin router group. Kept separate from RegisterMediaRoutes and
// RegisterStickerPresignRoute so each presign flavor owns its own
// route + DTO + handler.
func RegisterBorderPresignRoute(rg *gin.RouterGroup, h *BorderPresignHandler, adminMiddleware gin.HandlerFunc) {
	media := rg.Group("/media")
	media.POST("/presign-border", adminMiddleware, h.GetPresignedBorderURL)
}
