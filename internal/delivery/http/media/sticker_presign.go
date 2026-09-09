package media

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"indagio-api/internal/delivery/http/utils"
	"indagio-api/internal/domain"
)

const (
	// StickerPresignedURLTTL is the lifetime of the presigned PUT URL.
	// 15 minutes matches the video presign endpoint.
	StickerPresignedURLTTL = 15 * 60

	// DefaultStickerMaxSizeBytes is the server-side cap that the FE
	// may not exceed. The presign endpoint clamps the caller's request
	// down to this value.
	DefaultStickerMaxSizeBytes int64 = 512 * 1024 // 512 KB
)

// StickerPresignDeps is the bundle of dependencies the sticker presign
// handler needs. Passed to NewStickerPresignHandler so the media
// package doesn't have to import the usecase package.
type StickerPresignDeps struct {
	Storage domain.StorageRepository
	Logger  *slog.Logger
	// LookupUserCounts returns the caller's current sticker count and
	// pack count. Implemented in main.go by composing the sticker repo.
	LookupUserCounts func(ctx interface{ Done() <-chan struct{} }, userID uuid.UUID) (stickers, packs int, err error)
	MaxStickers      int
	MaxPacks         int
}

// StickerPresignHandler issues presigned URLs for sticker uploads.
// It is a separate type from MediaHandler so the media package stays
// focused on avatars / threads / comments / chat, and the sticker
// upload flow is self-contained.
type StickerPresignHandler struct {
	storage          domain.StorageRepository
	logger           *slog.Logger
	lookupUserCounts func(userID uuid.UUID) (stickers, packs int, err error)
	maxStickers      int
	maxPacks         int
}

// NewStickerPresignHandler constructs the handler. The lookup callback
// is required so the response can include quota information.
func NewStickerPresignHandler(
	storage domain.StorageRepository,
	logger *slog.Logger,
	lookupUserCounts func(userID uuid.UUID) (stickers, packs int, err error),
	maxStickers int,
	maxPacks int,
) *StickerPresignHandler {
	return &StickerPresignHandler{
		storage:          storage,
		logger:           logger,
		lookupUserCounts: lookupUserCounts,
		maxStickers:      maxStickers,
		maxPacks:         maxPacks,
	}
}

// GetPresignedStickerURL — POST /api/media/presigned-sticker
//
// Validates the request, clamps the size to the server cap, and
// returns a presigned PUT URL plus the caller's current quota.
func (h *StickerPresignHandler) GetPresignedStickerURL(c *gin.Context) {
	var req PresignedStickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

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

	// Clamp the requested size to the server cap.
	effectiveMax := req.MaxSizeBytes
	if effectiveMax > DefaultStickerMaxSizeBytes {
		effectiveMax = DefaultStickerMaxSizeBytes
	}

	// Generate the raw key. The pending prefix matches the validation
	// in the sticker usecase's CreateSticker.
	fileUUID := uuid.New().String()
	rawKey := "stickers/pending/" + userID.String() + "/" + fileUUID + "." + req.Extension

	presignedURL, err := h.storage.GetPresignedUploadURL(c.Request.Context(), rawKey, req.ContentType)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	// Best-effort quota lookup. A failure here is non-fatal: the
	// client still gets a valid upload URL.
	stickersUsed, packsUsed := 0, 0
	if h.lookupUserCounts != nil {
		n, p, err := h.lookupUserCounts(userID)
		if err != nil {
			h.logger.WarnContext(c.Request.Context(), "sticker presign: quota lookup failed",
				"user_id", userID, "error", err)
		} else {
			stickersUsed = n
			packsUsed = p
		}
	}

	c.JSON(http.StatusOK, PresignedStickerResponse{
		UploadURL:    presignedURL,
		RawKey:       rawKey,
		MaxSizeBytes: effectiveMax,
		MaxStickers:  h.maxStickers,
		MaxPacks:     h.maxPacks,
		StickersUsed: stickersUsed,
		PacksUsed:    packsUsed,
	})
}

// RegisterStickerPresignRoute mounts the sticker presign endpoint on
// the media router group. Kept separate from RegisterMediaRoutes so
// the media handler stays focused on the legacy endpoints.
func RegisterStickerPresignRoute(rg *gin.RouterGroup, h *StickerPresignHandler, authMiddleware gin.HandlerFunc) {
	media := rg.Group("/media")
	media.POST("/presigned-sticker", authMiddleware, h.GetPresignedStickerURL)
}
