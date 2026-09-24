package instrument_scoring

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"indagio-api/internal/delivery/http/utils"
	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InstrumentScoringHandler struct {
	uc     domain.InstrumentScoringUsecase
	logger *slog.Logger
}

func NewInstrumentScoringHandler(uc domain.InstrumentScoringUsecase, logger *slog.Logger) *InstrumentScoringHandler {
	return &InstrumentScoringHandler{uc: uc, logger: logger}
}

// PublishVersion godoc
// @Summary		Publish instrument scoring catalog
// @Description	Publishes a new versioned set of items, subscales, scoring rules and bands for an instrument
// @Tags		Instrument Scoring
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		instrumentID path string true "Instrument ID"
// @Param		body body PublishVersionRequest true "Catalog payload"
// @Success		201 {object} CatalogResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Router		/projects/{id}/instruments/{instrumentID}/versions [post]
// @Security	BearerAuth
func (h *InstrumentScoringHandler) PublishVersion(c *gin.Context) {
	var req PublishVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	items, subscales, rules, bands := req.ToDomain()
	catalog, err := h.uc.PublishVersion(ctx, actorID, instrumentID, items, subscales, rules, bands)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, ToCatalogResponse(catalog))
}

// GetCatalog godoc
// @Summary		Get instrument scoring catalog
// @Tags		Instrument Scoring
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		instrumentID path string true "Instrument ID"
// @Param		version query int false "Version (defaults to latest)"
// @Success		200 {object} CatalogResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		404 {object} map[string]string
// @Router		/projects/{id}/instruments/{instrumentID}/versions [get]
// @Security	BearerAuth
func (h *InstrumentScoringHandler) GetCatalog(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	var version *int
	if raw := c.Query("version"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
			return
		}
		version = &v
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	catalog, err := h.uc.GetCatalog(ctx, actorID, instrumentID, version)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ToCatalogResponse(catalog))
}

// ListVersions godoc
// @Summary		List published catalog versions for an instrument
// @Tags		Instrument Scoring
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		instrumentID path string true "Instrument ID"
// @Success		200 {array} int
// @Router		/projects/{id}/instruments/{instrumentID}/versions/list [get]
// @Security	BearerAuth
func (h *InstrumentScoringHandler) ListVersions(c *gin.Context) {
	actorID, ok := actorIDFromContext(c)
	if !ok {
		return
	}
	instrumentID, err := uuid.Parse(c.Param("instrumentID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid instrument id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	versions, err := h.uc.ListVersions(ctx, actorID, instrumentID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, versions)
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

func RegisterInstrumentScoringRoutes(rg *gin.RouterGroup, h *InstrumentScoringHandler, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/projects/:id/instruments/:instrumentID/versions")
	group.Use(authMiddleware)
	group.POST("", h.PublishVersion)
	group.GET("", h.GetCatalog)
	group.GET("/list", h.ListVersions)
}
