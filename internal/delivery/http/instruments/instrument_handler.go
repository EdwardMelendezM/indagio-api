package instruments

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

type InstrumentHandler struct {
	uc     domain.InstrumentUsecase
	logger *slog.Logger
}

func NewInstrumentHandler(uc domain.InstrumentUsecase, logger *slog.Logger) *InstrumentHandler {
	return &InstrumentHandler{uc: uc, logger: logger}
}

type createInstrumentRequest struct {
	Name   string          `json:"name" binding:"required,min=2,max=160"`
	Kind   string          `json:"kind" binding:"required"`
	Config json.RawMessage `json:"config"`
}

type updateInstrumentRequest struct {
	Name   *string          `json:"name"`
	Config *json.RawMessage `json:"config"`
	Status *string          `json:"status"`
}

type createQuestionnaireRequest struct {
	Name   string          `json:"name" binding:"required,min=2,max=160"`
	Schema json.RawMessage `json:"schema" binding:"required"`
}

type instrumentResponse struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Config    json.RawMessage `json:"config"`
	Version   int             `json:"version"`
	Status    string          `json:"status"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type questionnaireResponse struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Version   int             `json:"version"`
	Status    string          `json:"status"`
	Schema    json.RawMessage `json:"schema"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func toInstrumentResponse(inst *domain.Instrument) instrumentResponse {
	return instrumentResponse{
		ID:        inst.ID.String(),
		ProjectID: inst.ProjectID.String(),
		Name:      inst.Name,
		Kind:      string(inst.Kind),
		Config:    inst.Config,
		Version:   inst.Version,
		Status:    string(inst.Status),
		CreatedBy: inst.CreatedBy.String(),
		CreatedAt: inst.CreatedAt,
		UpdatedAt: inst.UpdatedAt,
	}
}

func toQuestionnaireResponse(q *domain.Questionnaire) questionnaireResponse {
	return questionnaireResponse{
		ID:        q.ID.String(),
		ProjectID: q.ProjectID.String(),
		Name:      q.Name,
		Version:   q.Version,
		Status:    q.Status,
		Schema:    q.Schema,
		CreatedBy: q.CreatedBy.String(),
		CreatedAt: q.CreatedAt,
		UpdatedAt: q.UpdatedAt,
	}
}

// CreateInstrument godoc
// @Summary		Create instrument
// @Description	Create an instrument template in a project
// @Tags		Instruments
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body map[string]interface{} true "Instrument payload"
// @Success		201 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/instruments [post]
// @Security	BearerAuth
func (h *InstrumentHandler) CreateInstrument(c *gin.Context) {
	var req createInstrumentRequest
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
	instrument, err := h.uc.CreateInstrument(ctx, userID, projectID, req.Name, req.Kind, req.Config)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, toInstrumentResponse(instrument))
}

// ListInstruments godoc
// @Summary		List instruments
// @Description	List instrument templates in a project
// @Tags		Instruments
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {array} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/instruments [get]
// @Security	BearerAuth
func (h *InstrumentHandler) ListInstruments(c *gin.Context) {
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
	instruments, err := h.uc.ListInstruments(ctx, userID, projectID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]instrumentResponse, 0, len(instruments))
	for _, item := range instruments {
		resp = append(resp, toInstrumentResponse(&item))
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateInstrument godoc
// @Summary		Update instrument
// @Description	Update instrument name, config, or status
// @Tags		Instruments
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		instrumentID path string true "Instrument ID"
// @Param		body body map[string]interface{} true "Instrument patch payload"
// @Success		200 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/instruments/{instrumentID} [patch]
// @Security	BearerAuth
func (h *InstrumentHandler) UpdateInstrument(c *gin.Context) {
	var req updateInstrumentRequest
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
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var status *domain.InstrumentStatus
	if req.Status != nil {
		parsed := domain.InstrumentStatus(*req.Status)
		status = &parsed
	}
	instrument, err := h.uc.UpdateInstrument(ctx, userID, instrumentID, req.Name, req.Config, status)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toInstrumentResponse(instrument))
}

// CreateQuestionnaire godoc
// @Summary		Create questionnaire
// @Description	Create a questionnaire schema for a project
// @Tags		Questionnaires
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body map[string]interface{} true "Questionnaire payload"
// @Success		201 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/questionnaires [post]
// @Security	BearerAuth
func (h *InstrumentHandler) CreateQuestionnaire(c *gin.Context) {
	var req createQuestionnaireRequest
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
	questionnaire, err := h.uc.CreateQuestionnaire(ctx, userID, projectID, req.Name, req.Schema)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, toQuestionnaireResponse(questionnaire))
}

// ListQuestionnaires godoc
// @Summary		List questionnaires
// @Description	List questionnaire versions in a project
// @Tags		Questionnaires
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {array} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/questionnaires [get]
// @Security	BearerAuth
func (h *InstrumentHandler) ListQuestionnaires(c *gin.Context) {
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
	questionnaires, err := h.uc.ListQuestionnaires(ctx, userID, projectID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]questionnaireResponse, 0, len(questionnaires))
	for _, item := range questionnaires {
		resp = append(resp, toQuestionnaireResponse(&item))
	}
	c.JSON(http.StatusOK, resp)
}

// PublishQuestionnaire godoc
// @Summary		Publish questionnaire
// @Description	Publish a questionnaire version for active use
// @Tags		Questionnaires
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		questionnaireID path string true "Questionnaire ID"
// @Success		200 {object} map[string]interface{}
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		404 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/questionnaires/{questionnaireID}/publish [post]
// @Security	BearerAuth
func (h *InstrumentHandler) PublishQuestionnaire(c *gin.Context) {
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
	questionnaireID, err := uuid.Parse(c.Param("questionnaireID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid questionnaire id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	questionnaire, err := h.uc.PublishQuestionnaire(ctx, userID, questionnaireID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toQuestionnaireResponse(questionnaire))
}

func RegisterInstrumentRoutes(rg *gin.RouterGroup, h *InstrumentHandler, authMiddleware gin.HandlerFunc) {
	projectInstruments := rg.Group("/projects/:id/instruments")
	projectInstruments.Use(authMiddleware)
	projectInstruments.POST("", h.CreateInstrument)
	projectInstruments.GET("", h.ListInstruments)
	projectInstruments.PATCH("/:instrumentID", h.UpdateInstrument)

	projectQuestionnaires := rg.Group("/projects/:id/questionnaires")
	projectQuestionnaires.Use(authMiddleware)
	projectQuestionnaires.POST("", h.CreateQuestionnaire)
	projectQuestionnaires.GET("", h.ListQuestionnaires)
	projectQuestionnaires.POST("/:questionnaireID/publish", h.PublishQuestionnaire)
}
