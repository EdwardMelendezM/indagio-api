package participants

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"indagio-api/internal/delivery/http/utils"
	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ParticipantHandler struct {
	uc     domain.ParticipantUsecase
	logger *slog.Logger
}

func NewParticipantHandler(uc domain.ParticipantUsecase, logger *slog.Logger) *ParticipantHandler {
	return &ParticipantHandler{uc: uc, logger: logger}
}

type createParticipantRequest struct {
	DisplayName        string `json:"display_name" binding:"required,min=2,max=160"`
	ExternalIdentifier string `json:"external_identifier"`
}

type updateParticipantStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type participantResponse struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	Code               string    `json:"code"`
	DisplayName        string    `json:"display_name"`
	ExternalIdentifier string    `json:"external_identifier,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func toParticipantResponse(p *domain.Participant) participantResponse {
	return participantResponse{
		ID:                 p.ID.String(),
		ProjectID:          p.ProjectID.String(),
		Code:               p.Code,
		DisplayName:        p.DisplayName,
		ExternalIdentifier: p.ExternalIdentifier,
		Status:             string(p.Status),
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

func (h *ParticipantHandler) Create(c *gin.Context) {
	var req createParticipantRequest
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
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	participant, err := h.uc.CreateParticipant(ctx, userID, projectID, req.DisplayName, req.ExternalIdentifier)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, toParticipantResponse(participant))
}

func (h *ParticipantHandler) List(c *gin.Context) {
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
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	participants, err := h.uc.ListParticipants(ctx, userID, projectID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]participantResponse, 0, len(participants))
	for _, p := range participants {
		resp = append(resp, toParticipantResponse(&p))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ParticipantHandler) GetByCode(c *gin.Context) {
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
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	participant, err := h.uc.GetParticipantByCode(ctx, userID, projectID, c.Param("code"))
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toParticipantResponse(participant))
}

func (h *ParticipantHandler) UpdateStatus(c *gin.Context) {
	var req updateParticipantStatusRequest
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
	participantID, err := uuid.Parse(c.Param("participantID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	status := domain.ParticipantStatus(req.Status)
	participant, err := h.uc.UpdateParticipantStatus(ctx, userID, participantID, status)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toParticipantResponse(participant))
}

func RegisterParticipantRoutes(rg *gin.RouterGroup, h *ParticipantHandler, authMiddleware gin.HandlerFunc) {
	participants := rg.Group("/projects/:id/participants")
	participants.Use(authMiddleware)
	participants.POST("", h.Create)
	participants.GET("", h.List)
	participants.GET(":code", h.GetByCode)
	participants.PATCH(":participantID/status", h.UpdateStatus)
}
