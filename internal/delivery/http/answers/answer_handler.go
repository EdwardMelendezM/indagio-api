package answers

import (
	"context"
	"encoding/json"
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

type createAnswerRequest struct {
	ParticipantID     string          `json:"participant_id" binding:"required"`
	InstrumentID      *string         `json:"instrument_id"`
	QuestionnaireID   *string         `json:"questionnaire_id"`
	QuestionKey       string          `json:"question_key" binding:"required"`
	AnswerType        string          `json:"answer_type" binding:"required"`
	Value             json.RawMessage `json:"value" binding:"required"`
	ClientGeneratedID *string         `json:"client_generated_id"`
}

type createMediaRequest struct {
	AnswerID        *string `json:"answer_id"`
	FileKey         string  `json:"file_key" binding:"required"`
	MimeType        string  `json:"mime_type" binding:"required"`
	SizeBytes       int64   `json:"size_bytes" binding:"required"`
	DurationSeconds *int    `json:"duration_seconds"`
	Checksum        *string `json:"checksum"`
}

type answerResponse struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	ParticipantID     string          `json:"participant_id"`
	InstrumentID      *string         `json:"instrument_id,omitempty"`
	QuestionnaireID   *string         `json:"questionnaire_id,omitempty"`
	QuestionKey       string          `json:"question_key"`
	AnswerType        string          `json:"answer_type"`
	Value             json.RawMessage `json:"value"`
	Status            string          `json:"status"`
	SyncStatus        string          `json:"sync_status"`
	ClientGeneratedID *string         `json:"client_generated_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type mediaResponse struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	ParticipantID   string    `json:"participant_id"`
	AnswerID        *string   `json:"answer_id,omitempty"`
	FileKey         string    `json:"file_key"`
	MimeType        string    `json:"mime_type"`
	SizeBytes       int64     `json:"size_bytes"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	Checksum        *string   `json:"checksum,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func toAnswerResponse(a *domain.AnswerRecord) answerResponse {
	resp := answerResponse{
		ID:            a.ID.String(),
		ProjectID:     a.ProjectID.String(),
		ParticipantID: a.ParticipantID.String(),
		QuestionKey:   a.QuestionKey,
		AnswerType:    string(a.AnswerType),
		Value:         a.Value,
		Status:        string(a.Status),
		SyncStatus:    string(a.SyncStatus),
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
	if a.InstrumentID != nil {
		v := a.InstrumentID.String()
		resp.InstrumentID = &v
	}
	if a.QuestionnaireID != nil {
		v := a.QuestionnaireID.String()
		resp.QuestionnaireID = &v
	}
	if a.ClientGeneratedID != nil {
		resp.ClientGeneratedID = a.ClientGeneratedID
	}
	return resp
}

func toMediaResponse(m *domain.MediaFile) mediaResponse {
	resp := mediaResponse{
		ID:            m.ID.String(),
		ProjectID:     m.ProjectID.String(),
		ParticipantID: m.ParticipantID.String(),
		FileKey:       m.FileKey,
		MimeType:      m.MimeType,
		SizeBytes:     m.SizeBytes,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.AnswerID != nil {
		v := m.AnswerID.String()
		resp.AnswerID = &v
	}
	if m.DurationSeconds != nil {
		resp.DurationSeconds = m.DurationSeconds
	}
	if m.Checksum != nil {
		resp.Checksum = m.Checksum
	}
	return resp
}

// CreateAnswer godoc
// @Summary		Create answer
// @Description	Create a participant answer record in a project
// @Tags		Answers
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body map[string]interface{} true "Answer payload"
// @Success		201 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/answers [post]
// @Security	BearerAuth
func (h *AnswerHandler) CreateAnswer(c *gin.Context) {
	var req createAnswerRequest
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
	c.JSON(http.StatusCreated, toAnswerResponse(answer))
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
	resp := make([]answerResponse, 0, len(answers))
	for _, a := range answers {
		resp = append(resp, toAnswerResponse(&a))
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
// @Param		body body createMediaRequest true "Media payload"
// @Success		201 {object} mediaResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/participants/{participantID}/media [post]
// @Security	BearerAuth
func (h *AnswerHandler) CreateMedia(c *gin.Context) {
	var req createMediaRequest
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
	c.JSON(http.StatusCreated, toMediaResponse(media))
}

func RegisterAnswerRoutes(rg *gin.RouterGroup, h *AnswerHandler, authMiddleware gin.HandlerFunc) {
	answers := rg.Group("/projects/:id")
	answers.Use(authMiddleware)
	answers.POST("/answers", h.CreateAnswer)
	answers.GET("/participants/:participantID/answers", h.ListAnswers)
	answers.POST("/participants/:participantID/media", h.CreateMedia)
}
