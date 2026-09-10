package exports

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExportHandler struct {
	uc domain.ExportUsecase
}

func NewExportHandler(uc domain.ExportUsecase) *ExportHandler {
	return &ExportHandler{uc: uc}
}

type requestExportRequest struct {
	Format  string          `json:"format" binding:"required"`
	Options json.RawMessage `json:"options"`
}

type exportResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	CreatedBy   string     `json:"created_by"`
	Format      string     `json:"format"`
	Status      string     `json:"status"`
	FileKey     *string    `json:"file_key,omitempty"`
	ErrorLog    *string    `json:"error_log,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func toExportResponse(item *domain.ProjectExport) exportResponse {
	resp := exportResponse{
		ID:        item.ID.String(),
		ProjectID: item.ProjectID.String(),
		CreatedBy: item.CreatedBy.String(),
		Format:    string(item.Format),
		Status:    string(item.Status),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	if item.FileKey != nil {
		resp.FileKey = item.FileKey
	}
	if item.ErrorLog != nil {
		resp.ErrorLog = item.ErrorLog
	}
	if item.CompletedAt != nil {
		resp.CompletedAt = item.CompletedAt
	}
	return resp
}

func actorIDFromContext(c *gin.Context) uuid.UUID {
	if value, exists := c.Get("userID"); exists {
		switch v := value.(type) {
		case uuid.UUID:
			return v
		case string:
			parsed, err := uuid.Parse(v)
			if err == nil {
				return parsed
			}
		}
	}
	header := c.GetHeader("X-Actor-ID")
	if header == "" {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(header)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func (h *ExportHandler) RequestExport(c *gin.Context) {
	var req requestExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	actorID := actorIDFromContext(c)
	if actorID == uuid.Nil {
		actorID = uuid.New()
	}
	item, err := h.uc.RequestExport(c.Request.Context(), actorID, projectID, req.Format, req.Options)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusAccepted, toExportResponse(item))
}

func (h *ExportHandler) ListExports(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	actorID := actorIDFromContext(c)
	if actorID == uuid.Nil {
		actorID = uuid.New()
	}
	items, err := h.uc.ListExports(c.Request.Context(), actorID, projectID)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	resp := make([]exportResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toExportResponse(&items[i]))
	}
	c.JSON(http.StatusOK, resp)
}

func RegisterExportRoutes(rg *gin.RouterGroup, h *ExportHandler) {
	projects := rg.Group("/projects")
	projects.POST("/:id/exports", h.RequestExport)
	projects.GET("/:id/exports", h.ListExports)
}
