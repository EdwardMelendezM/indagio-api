package sync

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SyncHandler struct {
	uc domain.SyncUsecase
}

func NewSyncHandler(uc domain.SyncUsecase) *SyncHandler {
	return &SyncHandler{uc: uc}
}

type submitBatchRequest struct {
	ProjectID              string          `json:"project_id" binding:"required"`
	ParticipantID          string          `json:"participant_id" binding:"required"`
	ClientGeneratedBatchID string          `json:"client_generated_batch_id" binding:"required"`
	Payload                json.RawMessage `json:"payload" binding:"required"`
}

type syncBatchResponse struct {
	ID                     string    `json:"id"`
	ProjectID              string    `json:"project_id"`
	ParticipantID          string    `json:"participant_id"`
	ClientGeneratedBatchID string    `json:"client_generated_batch_id"`
	Status                 string    `json:"status"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func toSyncBatchResponse(batch *domain.SyncBatch) syncBatchResponse {
	return syncBatchResponse{
		ID:                     batch.ID.String(),
		ProjectID:              batch.ProjectID.String(),
		ParticipantID:          batch.ParticipantID.String(),
		ClientGeneratedBatchID: batch.ClientGeneratedBatchID,
		Status:                 string(batch.Status),
		CreatedAt:              batch.CreatedAt,
		UpdatedAt:              batch.UpdatedAt,
	}
}

func (h *SyncHandler) SubmitBatch(c *gin.Context) {
	var req submitBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "validation failed"})
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}
	participantID, err := uuid.Parse(req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}
	actorID, err := uuid.Parse(c.GetHeader("X-Actor-ID"))
	if err != nil || actorID == uuid.Nil {
		actorID = uuid.New() // temporary compatibility for unscoped local validation
	}
	batch, err := h.uc.SubmitBatch(c.Request.Context(), actorID, projectID, participantID, req.ClientGeneratedBatchID, req.Payload)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, toSyncBatchResponse(batch))
}

func (h *SyncHandler) ListBatches(c *gin.Context) {
	participantID, err := uuid.Parse(c.Query("participant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}
	actorID, err := uuid.Parse(c.GetHeader("X-Actor-ID"))
	if err != nil || actorID == uuid.Nil {
		actorID = uuid.New()
	}
	batches, err := h.uc.ListBatches(c.Request.Context(), actorID, participantID)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	resp := make([]syncBatchResponse, 0, len(batches))
	for _, batch := range batches {
		resp = append(resp, toSyncBatchResponse(&batch))
	}
	c.JSON(http.StatusOK, resp)
}

func RegisterSyncRoutes(rg *gin.RouterGroup, h *SyncHandler) {
	group := rg.Group("/sync")
	group.POST("/batches", h.SubmitBatch)
	group.GET("/batches", h.ListBatches)
}
