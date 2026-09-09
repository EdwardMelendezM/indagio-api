package projects

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"indagio-api/internal/delivery/http/utils"
	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	uc     domain.ProjectUsecase
	logger *slog.Logger
}

func NewProjectHandler(uc domain.ProjectUsecase, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{uc: uc, logger: logger}
}

type createProjectRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=160"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type inviteMemberRequest struct {
	Email     string    `json:"email" binding:"required,email"`
	Role      string    `json:"role" binding:"required"`
	ExpiresAt time.Time `json:"expires_at" binding:"required"`
}

type projectResponse struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type projectMemberResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type projectInvitationResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func toProjectResponse(p *domain.Project) projectResponse {
	return projectResponse{
		ID:          p.ID.String(),
		OwnerID:     p.OwnerID.String(),
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toProjectMemberResponse(m domain.ProjectMember) projectMemberResponse {
	return projectMemberResponse{
		ID:        m.ID.String(),
		ProjectID: m.ProjectID.String(),
		UserID:    m.UserID.String(),
		Role:      string(m.Role),
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func toProjectInvitationResponse(i *domain.ProjectInvitation) projectInvitationResponse {
	return projectInvitationResponse{
		ID:        i.ID.String(),
		ProjectID: i.ProjectID.String(),
		Email:     i.Email,
		Role:      string(i.Role),
		Status:    i.Status,
		ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt,
	}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	project, err := h.uc.CreateProject(ctx, userID, req.Name, req.Description)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, toProjectResponse(project))
}

func (h *ProjectHandler) List(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var status *domain.ProjectStatus
	if raw := c.Query("status"); raw != "" {
		v := domain.ProjectStatus(raw)
		status = &v
	}

	projects, err := h.uc.ListProjects(ctx, userID, status)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, toProjectResponse(&p))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) GetByID(c *gin.Context) {
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

	project, err := h.uc.GetProject(ctx, userID, projectID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) Update(c *gin.Context) {
	var req updateProjectRequest
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

	var status *domain.ProjectStatus
	if req.Status != nil {
		parsed := domain.ProjectStatus(*req.Status)
		status = &parsed
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	project, err := h.uc.UpdateProject(ctx, userID, projectID, req.Name, req.Description, status)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) Archive(c *gin.Context) {
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

	if err := h.uc.ArchiveProject(ctx, userID, projectID); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "archived"})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
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

	if err := h.uc.DeleteProject(ctx, userID, projectID); err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProjectHandler) InviteMember(c *gin.Context) {
	var req inviteMemberRequest
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

	role := domain.ProjectMemberRole(req.Role)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	invitation, err := h.uc.InviteMember(ctx, userID, projectID, req.Email, role, req.ExpiresAt)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, toProjectInvitationResponse(invitation))
}

func (h *ProjectHandler) Members(c *gin.Context) {
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

	members, err := h.uc.ListMembers(ctx, userID, projectID)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	resp := make([]projectMemberResponse, 0, len(members))
	for _, member := range members {
		resp = append(resp, toProjectMemberResponse(member))
	}
	c.JSON(http.StatusOK, resp)
}

func RegisterProjectRoutes(rg *gin.RouterGroup, h *ProjectHandler, authMiddleware gin.HandlerFunc) {
	projects := rg.Group("/projects")
	projects.Use(authMiddleware)
	projects.POST("", h.Create)
	projects.GET("", h.List)
	projects.GET(":id", h.GetByID)
	projects.PATCH(":id", h.Update)
	projects.DELETE(":id", h.Delete)
	projects.POST(":id/archive", h.Archive)
	projects.POST(":id/members/invite", h.InviteMember)
	projects.GET(":id/members", h.Members)
}

func (h *ProjectHandler) HandleError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUserAlreadyExists) || errors.Is(err, domain.ErrConflict) {
		utils.HandleError(c, err, h.logger)
		return
	}
	utils.HandleError(c, err, h.logger)
}
