package answers

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

type AnswerHandler struct {
	uc     domain.AnswerUsecase
	logger *slog.Logger
}

func NewAnswerHandler(uc domain.AnswerUsecase, logger *slog.Logger) *AnswerHandler {
	return &AnswerHandler{uc: uc, logger: logger}
}

// CreateAnswer godoc
// @Summary		Create answer
// @Description	Create a participant answer record in a project
// @Tags		Answers
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body CreateAnswerRequest true "Answer payload"
// @Success		201 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/answers [post]
// @Security	BearerAuth
func (h *AnswerHandler) CreateAnswer(c *gin.Context) {
	var req CreateAnswerRequest
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
	participantID, err := uuid.Parse(req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}
	var instrumentID, questionnaireID *uuid.UUID
	if req.InstrumentID != nil {
		parsed, err := uuid.Parse(*req.InstrumentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
			return
		}
		instrumentID = &parsed
	}
	if req.QuestionnaireID != nil {
		parsed, err := uuid.Parse(*req.QuestionnaireID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid questionnaire id"})
			return
		}
		questionnaireID = &parsed
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	answer, err := h.uc.CreateAnswer(ctx, userID, projectID, participantID, instrumentID, questionnaireID, req.QuestionKey, req.AnswerType, req.Value, req.ClientGeneratedID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, ToAnswerResponse(answer))
}

// ListAnswers godoc
// @Summary		List participant answers
// @Description	List answers submitted by a participant in a project
// @Tags		Answers
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		participantID path string true "Participant ID"
// @Success		200 {array} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants/{participantID}/answers [get]
// @Security	BearerAuth
func (h *AnswerHandler) ListAnswers(c *gin.Context) {
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
	participantID, err := uuid.Parse(c.Param("participantID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	answers, err := h.uc.ListAnswers(ctx, userID, projectID, participantID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]AnswerResponse, 0, len(answers))
	for _, a := range answers {
		resp = append(resp, ToAnswerResponse(&a))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateMedia godoc
// @Summary		Create media record
// @Description	Register uploaded participant media metadata
// @Tags		Answers
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		participantID path string true "Participant ID"
// @Param		body body CreateMediaRequest true "Media payload"
// @Success		201 {object} MediaResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants/{participantID}/media [post]
// @Security	BearerAuth
func (h *AnswerHandler) CreateMedia(c *gin.Context) {
	var req CreateMediaRequest
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
	participantID, err := uuid.Parse(c.Param("participantID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}
	var answerID *uuid.UUID
	if req.AnswerID != nil {
		parsed, err := uuid.Parse(*req.AnswerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id"})
			return
		}
		answerID = &parsed
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	media, err := h.uc.CreateMedia(ctx, userID, projectID, participantID, answerID, req.FileKey, req.MimeType, req.SizeBytes, req.DurationSeconds, req.Checksum)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, ToMediaResponse(media))
}

func RegisterAnswerRoutes(rg *gin.RouterGroup, h *AnswerHandler, authMiddleware gin.HandlerFunc) {
	answers := rg.Group("/projects/:id")
	answers.Use(authMiddleware)
	answers.POST("/answers", h.CreateAnswer)
	answers.GET("/participants/:participantID/answers", h.ListAnswers)
	answers.POST("/participants/:participantID/media", h.CreateMedia)
}
