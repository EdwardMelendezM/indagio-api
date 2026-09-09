package media

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"indagio-api/internal/delivery/http/utils"
	"indagio-api/internal/domain"
)

type MediaHandler struct {
	storageRepo         domain.StorageRepository
	publicDomain        string
	logger              *slog.Logger
	maxVideoSizeBytes   int64 // server-side cap for video uploads (from config)
	maxVideoDurationSec int   // server-side cap for video duration (from config)
}

func NewMediaHandler(sr domain.StorageRepository, logger *slog.Logger, publicDomain string, maxVideoSizeBytes int64, maxVideoDurationSec int) *MediaHandler {
	return &MediaHandler{
		storageRepo:         sr,
		publicDomain:        publicDomain,
		logger:              logger,
		maxVideoSizeBytes:   maxVideoSizeBytes,
		maxVideoDurationSec: maxVideoDurationSec,
	}
}

// RequestUploadURL godoc
// @Summary		Request presigned upload URL
// @Description	Get a presigned URL for uploading a file to Cloudflare R2. Return both upload URL and final file URL.
// @Tags		Media
// @Accept		json
// @Produce		json
// @Param		body body PresignUploadRequest true "Upload request with file metadata"
// @Success		200 {object} PresignUploadResponse "Presigned URL and final file URL"
// @Failure		400 {object} map[string]string "Invalid request body"
// @Failure		401 {object} map[string]string "Authentication required"
// @Failure		422 {object} map[string]string "Validation failed"
// @Failure		500 {object} map[string]string "Failed to generate upload URL"
// @Router		/media/presign [post]
// @Security	BearerAuth
func (h *MediaHandler) RequestUploadURL(c *gin.Context) {
	var req PresignUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return
	}

	fileUUID := uuid.New().String()
	var prefix string

	switch req.TargetType {
	case "avatar":
		prefix = fmt.Sprintf("users/%s/avatar", userID.String())
	case "thread":
		prefix = fmt.Sprintf("threads/pending-%s/images", userID.String())
	case "comment":
		prefix = fmt.Sprintf("comments/pending-%s", userID.String())
	case "chat":
		prefix = fmt.Sprintf("chat/pending-%s", userID.String())
	}

	key := fmt.Sprintf("%s/%s.%s", prefix, fileUUID, req.Extension)

	uploadURL, err := h.storageRepo.GetPresignedUploadURL(c.Request.Context(), key, req.ContentType)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	finalFileURL := fmt.Sprintf("%s/%s", h.publicDomain, key)

	c.JSON(http.StatusOK, PresignUploadResponse{
		UploadURL: uploadURL,
		FileURL:   finalFileURL,
	})
}

// GetPresignedVideoURL godoc
// @Summary		Request presigned URL for video upload
// @Description	Get a presigned PUT URL for uploading a video directly to R2. Returns upload URL, raw key, and the effective max size in bytes.
// @Tags		Media
// @Accept		json
// @Produce		json
// @Param		body body PresignedVideoRequest true "Video upload request with content type and max size"
// @Success		200 {object} PresignedVideoResponse "Presigned PUT URL, raw key, and effective max size"
// @Failure		400 {object} map[string]string "Invalid request body"
// @Failure		401 {object} map[string]string "Authentication required"
// @Failure		422 {object} map[string]string "Validation failed (content_type or max_size_bytes)"
// @Failure		500 {object} map[string]string "Failed to generate upload URL"
// @Router		/media/presigned-video [post]
// @Security	BearerAuth
func (h *MediaHandler) GetPresignedVideoURL(c *gin.Context) {
	var req PresignedVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Enforce server-side cap. If client requests more than the server allows,
	// clamp down to the server limit. This is the authoritative number.
	effectiveMax := req.MaxSizeBytes
	if effectiveMax > h.maxVideoSizeBytes {
		effectiveMax = h.maxVideoSizeBytes
	}

	rawKey := fmt.Sprintf("videos/raw/%s.mp4", uuid.New().String())

	presignedURL, err := h.storageRepo.GetPresignedPutURL(c.Request.Context(), rawKey, int(15*time.Minute/time.Second))
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, PresignedVideoResponse{
		UploadURL:      presignedURL,
		RawKey:         rawKey,
		MaxSizeBytes:   effectiveMax,
		MaxDurationSec: h.maxVideoDurationSec,
	})
}

func RegisterMediaRoutes(rg *gin.RouterGroup, h *MediaHandler, authMiddleware gin.HandlerFunc) {
	media := rg.Group("/media")
	{
		media.POST("/presign", authMiddleware, h.RequestUploadURL)
		media.POST("/presigned-video", authMiddleware, h.GetPresignedVideoURL)
	}
}
