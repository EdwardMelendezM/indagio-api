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

// Create godoc
// @Summary		Create participant
// @Description	Create a participant in a project
// @Tags		Participants
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body CreateParticipantRequest true "Participant payload"
// @Success		201 {object} ParticipantResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants [post]
// @Security	BearerAuth
func (h *ParticipantHandler) Create(c *gin.Context) {
	var req CreateParticipantRequest
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
	c.JSON(http.StatusCreated, ToParticipantResponse(participant))
}

// List godoc
// @Summary		List participants
// @Description	List participants for a project
// @Tags		Participants
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {array} ParticipantResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants [get]
// @Security	BearerAuth
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
	resp := make([]ParticipantResponse, 0, len(participants))
	for _, p := range participants {
		resp = append(resp, ToParticipantResponse(&p))
	}
	c.JSON(http.StatusOK, resp)
}

// GetByCode godoc
// @Summary		Get participant by code
// @Description	Get one participant by project-scoped code
// @Tags		Participants
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		code path string true "Participant code"
// @Success		200 {object} ParticipantResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		404 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants/code/{code} [get]
// @Security	BearerAuth
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
	c.JSON(http.StatusOK, ToParticipantResponse(participant))
}

// UpdateStatus godoc
// @Summary		Update participant status
// @Description	Update participant status (for example active, paused, archived)
// @Tags		Participants
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		participantID path string true "Participant ID"
// @Param		body body UpdateParticipantStatusRequest true "Status payload"
// @Success		200 {object} ParticipantResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants/{participantID}/status [patch]
// @Security	BearerAuth
func (h *ParticipantHandler) UpdateStatus(c *gin.Context) {
	var req UpdateParticipantStatusRequest
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
	c.JSON(http.StatusOK, ToParticipantResponse(participant))
}

func RegisterParticipantRoutes(rg *gin.RouterGroup, h *ParticipantHandler, authMiddleware gin.HandlerFunc) {
	participants := rg.Group("/projects/:id/participants")
	participants.Use(authMiddleware)
	participants.POST("", h.Create)
	participants.GET("", h.List)
	participants.GET("/code/:code", h.GetByCode)
	participants.PATCH("/:participantID/status", h.UpdateStatus)
}
