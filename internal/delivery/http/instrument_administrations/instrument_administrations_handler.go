package instrument_administrations

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

type InstrumentAdministrationHandler struct {
	uc     domain.InstrumentAdministrationUsecase
	logger *slog.Logger
}

func NewInstrumentAdministrationHandler(uc domain.InstrumentAdministrationUsecase, logger *slog.Logger) *InstrumentAdministrationHandler {
	return &InstrumentAdministrationHandler{uc: uc, logger: logger}
}

// StartAdministration godoc
// @Summary		Start an instrument administration
// @Tags		Instrument Administrations
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		participantID path string true "Participant ID"
// @Param		instrumentID path string true "Instrument ID"
// @Param		body body StartAdministrationRequest false "Idempotency payload"
// @Success		201 {object} AdministrationResponse
// @Router		/projects/{id}/participants/{participantID}/instruments/{instrumentID}/administrations [post]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) StartAdministration(c *gin.Context) {
	var req StartAdministrationRequest
	_ = c.ShouldBindJSON(&req) // body is optional; ignore empty-body bind errors
	actorID, ok := actorIDFromContext(c)
	if !ok {
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
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	admin, err := h.uc.StartAdministration(ctx, actorID, projectID, participantID, instrumentID, req.ClientGeneratedID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, ToAdministrationResponse(admin))
}

// ListAdministrations godoc
// @Summary		List administrations of an instrument for a participant
// @Tags		Instrument Administrations
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		participantID path string true "Participant ID"
// @Param		instrumentID path string true "Instrument ID"
// @Success		200 {array} AdministrationResponse
// @Router		/projects/{id}/participants/{participantID}/instruments/{instrumentID}/administrations [get]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) ListAdministrations(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	participantID, err := uuid.Parse(c.Param("participantID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	admins, err := h.uc.ListAdministrations(ctx, actorID, participantID, instrumentID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]AdministrationResponse, 0, len(admins))
	for _, a := range admins {
		resp = append(resp, ToAdministrationResponse(&a))
	}
	c.JSON(http.StatusOK, resp)
}

// CompleteAdministration godoc
// @Summary		Mark an administration as completed and compute its scores
// @Tags		Instrument Administrations
// @Produce		json
// @Param		administrationID path string true "Administration ID"
// @Success		200 {object} CompleteAdministrationResponse
// @Router		/administrations/{administrationID}/complete [patch]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) CompleteAdministration(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	administrationID, err := uuid.Parse(c.Param("administrationID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid administration id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	admin, scores, err := h.uc.CompleteAdministration(ctx, actorID, administrationID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, CompleteAdministrationResponse{Administration: ToAdministrationResponse(admin), Scores: ToScoreResponses(scores)})
}

// AbandonAdministration godoc
// @Summary		Mark an administration as abandoned
// @Tags		Instrument Administrations
// @Produce		json
// @Param		administrationID path string true "Administration ID"
// @Success		200 {object} AdministrationResponse
// @Router		/administrations/{administrationID}/abandon [patch]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) AbandonAdministration(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	administrationID, err := uuid.Parse(c.Param("administrationID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid administration id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	admin, err := h.uc.AbandonAdministration(ctx, actorID, administrationID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ToAdministrationResponse(admin))
}

// GetScores godoc
// @Summary		Get computed scores for an administration
// @Tags		Instrument Administrations
// @Produce		json
// @Param		administrationID path string true "Administration ID"
// @Success		200 {array} ScoreResponse
// @Router		/administrations/{administrationID}/scores [get]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) GetScores(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	administrationID, err := uuid.Parse(c.Param("administrationID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid administration id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	scores, err := h.uc.GetScores(ctx, actorID, administrationID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ToScoreResponses(scores))
}

// RecomputeScores godoc
// @Summary		Recompute scores for an administration (e.g. after fixing an answer)
// @Tags		Instrument Administrations
// @Produce		json
// @Param		administrationID path string true "Administration ID"
// @Success		200 {array} ScoreResponse
// @Router		/administrations/{administrationID}/scores/recompute [post]
// @Security	BearerAuth
func (h *InstrumentAdministrationHandler) RecomputeScores(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	administrationID, err := uuid.Parse(c.Param("administrationID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid administration id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	scores, err := h.uc.RecomputeScores(ctx, actorID, administrationID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ToScoreResponses(scores))
}

func actorIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return uuid.Nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return uuid.Nil, false
	}
	return userID, true
}

func RegisterInstrumentAdministrationRoutes(rg *gin.RouterGroup, h *InstrumentAdministrationHandler, authMiddleware gin.HandlerFunc) {
	scoped := rg.Group("/projects/:id/participants/:participantID/instruments/:instrumentID/administrations")
	scoped.Use(authMiddleware)
	scoped.POST("", h.StartAdministration)
	scoped.GET("", h.ListAdministrations)

	flat := rg.Group("/administrations")
	flat.Use(authMiddleware)
	flat.PATCH("/:administrationID/complete", h.CompleteAdministration)
	flat.PATCH("/:administrationID/abandon", h.AbandonAdministration)
	flat.GET("/:administrationID/scores", h.GetScores)
	flat.POST("/:administrationID/scores/recompute", h.RecomputeScores)
}
