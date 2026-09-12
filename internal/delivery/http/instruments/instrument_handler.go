package instruments

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

type InstrumentHandler struct {
	uc     domain.InstrumentUsecase
	logger *slog.Logger
}

func NewInstrumentHandler(uc domain.InstrumentUsecase, logger *slog.Logger) *InstrumentHandler {
	return &InstrumentHandler{uc: uc, logger: logger}
}

// CreateInstrument godoc
// @Summary		Create instrument
// @Description	Create an instrument template in a project
// @Tags		Instruments
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body CreateInstrumentRequest true "Instrument payload"
// @Success		201 {object} InstrumentResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/instruments [post]
// @Security	BearerAuth
func (h *InstrumentHandler) CreateInstrument(c *gin.Context) {
	var req CreateInstrumentRequest
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
	c.JSON(http.StatusCreated, ToInstrumentResponse(instrument))
}

// ListInstruments godoc
// @Summary		List instruments
// @Description	List instrument templates in a project
// @Tags		Instruments
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {array} InstrumentResponse
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
	resp := make([]InstrumentResponse, 0, len(instruments))
	for _, item := range instruments {
		resp = append(resp, ToInstrumentResponse(&item))
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
// @Param		body body UpdateInstrumentRequest true "Instrument patch payload"
// @Success		200 {object} InstrumentResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/instruments/{instrumentID} [patch]
// @Security	BearerAuth
func (h *InstrumentHandler) UpdateInstrument(c *gin.Context) {
	var req UpdateInstrumentRequest
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
	c.JSON(http.StatusOK, ToInstrumentResponse(instrument))
}

func RegisterInstrumentRoutes(rg *gin.RouterGroup, h *InstrumentHandler, authMiddleware gin.HandlerFunc) {
	projectInstruments := rg.Group("/projects/:id/instruments")
	projectInstruments.Use(authMiddleware)
	projectInstruments.POST("", h.CreateInstrument)
	projectInstruments.GET("", h.ListInstruments)
	projectInstruments.PATCH("/:instrumentID", h.UpdateInstrument)
}
