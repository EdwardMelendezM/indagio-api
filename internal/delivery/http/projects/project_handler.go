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

// Create godoc
// @Summary		Create project
// @Description	Create a new project owned by the authenticated user
// @Tags		Projects
// @Accept		json
// @Produce		json
// @Param		body body CreateProjectRequest true "Project payload"
// @Success		201 {object} ProjectResponse
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects [post]
// @Security	BearerAuth
func (h *ProjectHandler) Create(c *gin.Context) {
	var req CreateProjectRequest
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
	c.JSON(http.StatusCreated, ToProjectResponse(project))
}

// List godoc
// @Summary		List projects
// @Description	List projects where the authenticated user is owner or member
// @Tags		Projects
// @Produce		json
// @Param		status query string false "Filter by project status"
// @Success		200 {array} ProjectResponse
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects [get]
// @Security	BearerAuth
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
	resp := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, ToProjectResponse(&p))
	}
	c.JSON(http.StatusOK, resp)
}

// GetByID godoc
// @Summary		Get project by ID
// @Description	Get one project by ID when the authenticated user has access
// @Tags		Projects
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {object} ProjectResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		404 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id} [get]
// @Security	BearerAuth
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
	c.JSON(http.StatusOK, ToProjectResponse(project))
}

// Update godoc
// @Summary		Update project
// @Description	Update project name, description, or status
// @Tags		Projects
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body UpdateProjectRequest true "Project patch payload"
// @Success		200 {object} ProjectResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id} [patch]
// @Security	BearerAuth
func (h *ProjectHandler) Update(c *gin.Context) {
	var req UpdateProjectRequest
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
	c.JSON(http.StatusOK, ToProjectResponse(project))
}

// Archive godoc
// @Summary		Archive project
// @Description	Archive a project and make it read-only for normal operations
// @Tags		Projects
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {object} map[string]string
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/archive [post]
// @Security	BearerAuth
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

// Delete godoc
// @Summary		Delete project
// @Description	Delete a project permanently
// @Tags		Projects
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		204 {string} string "No Content"
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id} [delete]
// @Security	BearerAuth
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

// AddMember godoc
// @Summary		Add project member by user ID
// @Description	Directly add an existing user to a project as a project member
// @Tags		Projects
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body AddMemberRequest true "Member payload"
// @Success		201 {object} ProjectMemberResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		409 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/members [post]
// @Security	BearerAuth
func (h *ProjectHandler) AddMember(c *gin.Context) {
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid request"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	actorID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return
	}
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	memberUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	role := domain.ProjectMemberRole(req.Role)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	member, err := h.uc.AddMember(ctx, actorID, projectID, memberUserID, role)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, ToProjectMemberResponse(*member))
}

// InviteMember godoc
// @Summary		Invite project member
// @Description	Invite a user by email to join a project with a role
// @Tags		Projects
// @Accept		json
// @Produce		json
// @Param		id path string true "Project ID"
// @Param		body body InviteMemberRequest true "Invitation payload"
// @Success		201 {object} ProjectInvitationResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		422 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/members/invite [post]
// @Security	BearerAuth
func (h *ProjectHandler) InviteMember(c *gin.Context) {
	var req InviteMemberRequest
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
	c.JSON(http.StatusCreated, ToProjectInvitationResponse(invitation))
}

// Members godoc
// @Summary		List project members
// @Description	List all members in a project
// @Tags		Projects
// @Produce		json
// @Param		id path string true "Project ID"
// @Success		200 {array} ProjectMemberResponse
// @Failure		400 {object} map[string]string
// @Failure		401 {object} map[string]string
// @Failure		500 {object} map[string]string
// @Router		/projects/{id}/members [get]
// @Security	BearerAuth
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
	resp := make([]ProjectMemberResponse, 0, len(members))
	for _, member := range members {
		resp = append(resp, ToProjectMemberResponse(member))
	}
	c.JSON(http.StatusOK, resp)
}

func RegisterProjectRoutes(rg *gin.RouterGroup, h *ProjectHandler, authMiddleware gin.HandlerFunc) {
	projects := rg.Group("/projects")
	projects.Use(authMiddleware)
	projects.POST("", h.Create)
	projects.GET("", h.List)
	projects.GET("/:id", h.GetByID)
	projects.PATCH("/:id", h.Update)
	projects.DELETE("/:id", h.Delete)
	projects.POST("/:id/archive", h.Archive)
	projects.POST("/:id/members", h.AddMember)
	projects.POST("/:id/members/invite", h.InviteMember)
	projects.GET("/:id/members", h.Members)
}

func (h *ProjectHandler) HandleError(c *gin.Context, err error) {
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUserAlreadyExists) || errors.Is(err, domain.ErrConflict) {
		utils.HandleError(c, err, h.logger)
		return
	}
	utils.HandleError(c, err, h.logger)
}
