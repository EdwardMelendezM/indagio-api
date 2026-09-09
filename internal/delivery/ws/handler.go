package ws

import (
	"log/slog"
	"net/http"

	"foro-unsaac-backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for mobile app compatibility
	},
}

// WSHandler handles WebSocket upgrade requests
type WSHandler struct {
	hub      *Hub
	router   *Router
	tokenSvc domain.TokenService
	logger   *slog.Logger
}

// NewWSHandler creates a new WebSocket handler
func NewWSHandler(
	hub *Hub,
	router *Router,
	tokenSvc domain.TokenService,
	logger *slog.Logger,
) *WSHandler {
	return &WSHandler{
		hub:      hub,
		router:   router,
		tokenSvc: tokenSvc,
		logger:   logger,
	}
}

// HandleConnection upgrades HTTP to WebSocket
func (h *WSHandler) HandleConnection(c *gin.Context) {
	// Get token from query param (like SSE auth)
	token := c.Query("token")
	if token == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.tokenSvc.ValidateAccessToken(token)
	if err != nil {
		h.logger.Error("invalid ws token", "error", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	// Create client and register with hub
	client := NewClient(claims.UserID, conn, h.hub, h.logger)
	h.hub.Register() <- client

	// Start read/write pumps
	go client.writePump()
	go client.readPump(h.router)

	h.logger.Info("websocket client connected", "user_id", claims.UserID)
}

// RegisterWSRoutes registers the WebSocket routes
func RegisterWSRoutes(r *gin.Engine, h *WSHandler) {
	r.GET("/ws/chat", h.HandleConnection)
}
