package user

import (
	"log/slog"
	"net/http"

	"foro-unsaac-backend/internal/delivery/http/utils"
	"foro-unsaac-backend/internal/domain"
	userusecase "foro-unsaac-backend/internal/usecase/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user-related HTTP endpoints.
type UserHandler struct {
	authUC domain.AuthUsecase
	userUC *userusecase.UserUsecase
	logger *slog.Logger
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(authUC domain.AuthUsecase, userUC *userusecase.UserUsecase, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		authUC: authUC,
		userUC: userUC,
		logger: logger,
	}
}

// UpdateName godoc
// @Summary		Update user name
// @Description	Allows a user to update their own name or allows moderators/admins to update any user's name
// @Tags		Users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		id	path string true "User ID"
// @Param		body body UpdateUserNameRequest true "New name"
// @Success		200 {object} map[string]string "Name updated successfully"
// @Failure		400 {object} map[string]string "Invalid user id or request body"
// @Failure		401 {object} map[string]string "Unauthorized"
// @Failure		403 {object} map[string]string "Forbidden - not allowed to update this user"
// @Failure		422 {object} map[string]string "Validation failed - name must be 2-100 characters"
// @Failure		500 {object} map[string]string "Internal server error"
// @Router		/api/users/{id}/name [patch]
func (h *UserHandler) UpdateName(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		h.logger.Error("invalid user id type in context", "type", userIDVal)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return
	}

	var req UpdateUserNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.authUC.UpdateUserName(
		c.Request.Context(),
		userID,
		targetID,
		req.Name,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "name updated"})
}

// ListAvailableUsers godoc
// @Summary		List users available to start a conversation with
// @Description	Returns a paginated, searchable list of verified, non-deleted
// @Description	users (excluding the caller) that can be used as the
// @Description	target of POST /api/conversations.
// @Tags		Users
// @Accept		json
// @Produce		json
// @Security	BearerAuth
// @Param		page	query	int		false	"Page number (default 1)"
// @Param		limit	query	int		false	"Items per page (default 20, max 100)"
// @Param		q		query	string	false	"Case-insensitive search on name/email"
// @Success		200	{object}	map[string]interface{}	"Available users with pagination"
// @Failure		401	{object}	map[string]string	"Authentication required"
// @Failure		422	{object}	map[string]string	"Invalid query parameters"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router		/api/users/available [get]
func (h *UserHandler) ListAvailableUsers(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		h.logger.Error("invalid user id type in context", "type", userIDVal)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal context error"})
		return
	}

	var query ListAvailableUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid query parameters"})
		return
	}

	users, total, err := h.userUC.ListAvailable(
		c.Request.Context(),
		userID,
		query.Q,
		query.Page,
		query.Limit,
	)
	if err != nil {
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  ToUserResponseList(users),
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

// RegisterUserRoutes registers the user routes under /api/users.
//
// GET /:id is intentionally PUBLIC — it's the profile view used when
// another user opens someone else's profile page. It must NOT require
// auth, otherwise the FE has to bootstrap anonymous browsing with
// stub data. The other routes stay behind authMiddleware.
//
// Route ordering note: gin's radix tree gives static segments
// (e.g. "available") priority over wildcards (":id") regardless of
// registration order, so we don't have to be careful about the
// declaration sequence — but we keep the public route first to mirror
// the project's other public-on-:id patterns (see /users/:id/avatar).
func RegisterUserRoutes(rg *gin.RouterGroup, h *UserHandler, authMiddleware gin.HandlerFunc) {
	users := rg.Group("/users")
	{
		users.GET("/:id", h.GetPublicProfile) // public
		users.GET("/available", authMiddleware, h.ListAvailableUsers)
		users.PATCH("/:id/name", authMiddleware, h.UpdateName)
	}
}

// GetPublicProfile godoc
// @Summary		Get a user's public profile
// @Description	Returns the safe-to-expose fields of any user (id, name,
// @Description	role, avatar URL, selected border, created_at). Email and
// @Description	the internal `blocked` flag are intentionally omitted.
// @Description	No authentication required — this is the endpoint the
// @Description	frontend calls when rendering someone else's profile page.
// @Tags		Users
// @Produce		json
// @Param		id	path	string	true	"User ID (UUID)"
// @Success		200	{object}	PublicUserResponse	"Public profile"
// @Failure		404	{object}	map[string]string	"User not found"
// @Failure		422	{object}	map[string]string	"Invalid user id"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router		/api/users/{id} [get]
func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	rawID := c.Param("id")
	targetID, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.authUC.GetUserByID(c.Request.Context(), targetID)
	if err != nil {
		// AuthUsecase.GetUserByID wraps the repo error, so domain.ErrNotFound
		// is preserved underneath the fmt.Errorf — utils.HandleError maps
		// it to 404 via the AppError code.
		utils.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, ToPublicUserResponse(user))
}
