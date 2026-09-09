package chat_mvp

import (
	"log/slog"
	"net/http"
	"strings"

	"indagio-api/internal/delivery/ws"
	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// SubprotocolPrefix is the well-known prefix the FE prepends to the JWT
// when proposing the chat-mvp subprotocol on the upgrade request:
//
//	Sec-WebSocket-Protocol: chat-mvp.<jwt>
//
// The server extracts the token, validates it with the same TokenService
// the legacy /ws/chat handler uses, then upgrades echoing the chosen
// subprotocol so the FE's handshake promise is honoured.
const SubprotocolPrefix = "chat-mvp."

// newUpgrader is built per-request because the response header carries
// the exact token-derived subprotocol. gorilla's auto-negotiation does
// only exact-match (no prefix matching), so we leave Subprotocols nil
// and set the header manually — see HandleConnection.
func newUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // mobile app; same as legacy
		},
	}
}

// WSHandler serves the chat-mvp WebSocket endpoint at /api/messages/ws.
type WSHandler struct {
	hub      *ws.Hub
	router   *Router
	tokenSvc domain.TokenService
	logger   *slog.Logger
}

// NewWSHandler wires the chat-mvp WS handler. tokenSvc is the same
// TokenService used by the legacy /ws/chat — JWT validation is shared.
func NewWSHandler(
	hub *ws.Hub,
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

// HandleConnection performs the chat-mvp WS upgrade. Steps:
//
//  1. Read the `Sec-WebSocket-Protocol` header.
//  2. Find a token proposed as `chat-mvp.<jwt>`. Reject (400) if absent
//     or malformed — never silently downgrade to the legacy protocol.
//  3. Validate the JWT via the shared TokenService. 401 on failure.
//  4. Upgrade the connection, echoing the exact subprotocol the client
//     proposed (set manually — see newUpgrader comment).
//  5. Register a Client with Protocol="chat-mvp" and the larger frame
//     cap, then start the read/write pumps.
//
// The legacy `/ws/chat` endpoint is left untouched — both surfaces
// coexist.
func (h *WSHandler) HandleConnection(c *gin.Context) {
	rawProtocol, token, ok := extractChatMvpToken(c.GetHeader("Sec-WebSocket-Protocol"))
	if !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	claims, err := h.tokenSvc.ValidateAccessToken(token)
	if err != nil {
		h.logger.Info("chat-mvp ws: rejected (invalid token)", "error", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// gorilla's Upgrader.Subprotocols only does EXACT-MATCH against the
	// client-proposed values (verified against gorilla source). Our
	// token is dynamic, so we leave Subprotocols nil and set the
	// response header manually with the raw token-derived protocol.
	// Per RFC 6455 the server MUST echo one of the client's proposed
	// protocols — `chat-mvp.<jwt>` qualifies.
	hdr := http.Header{}
	hdr.Set("Sec-Websocket-Protocol", rawProtocol)

	upgrader := newUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, hdr)
	if err != nil {
		h.logger.Error("chat-mvp ws upgrade failed", "error", err)
		return
	}

	client := ws.NewClient(claims.UserID, conn, h.hub, h.logger)
	client.Protocol = "chat-mvp"
	client.MaxMessageSize = ws.ChatMvpMaxMessageSize
	h.hub.Register() <- client

	go client.WritePump()
	go client.ReadPump(h.router)

	h.logger.Info("chat-mvp websocket client connected",
		"user_id", claims.UserID,
		"protocol", client.Protocol,
	)
}

// extractChatMvpToken pulls the JWT out of a (possibly multi-value)
// Sec-WebSocket-Protocol header. Browsers send a comma-separated list;
// gorilla does not parse this for us.
//
// Returns the raw proposed subprotocol, the token it contained, and
// true on success. False if no proposal matching `chat-mvp.<token>`
// was found. The raw protocol is returned alongside the token so the
// caller can echo it back exactly in the upgrade response (gorilla's
// auto-negotiation cannot match a token-derived string without
// precomputing the Subprotocols list per-request).
func extractChatMvpToken(header string) (rawProtocol, token string, ok bool) {
	if header == "" {
		return "", "", false
	}
	for _, raw := range strings.Split(header, ",") {
		proto := strings.TrimSpace(raw)
		if strings.HasPrefix(proto, SubprotocolPrefix) {
			return proto, strings.TrimPrefix(proto, SubprotocolPrefix), true
		}
	}
	return "", "", false
}

// RegisterRoutes mounts the chat-mvp WS endpoint at /api/messages/ws.
// Auth happens in HandleConnection via the subprotocol JWT — no
// middleware on the route. Bearer tokens in headers/query are
// ignored here (chat-mvp uses the subprotocol exclusively).
func RegisterRoutes(r *gin.Engine, h *WSHandler) {
	r.GET("/api/messages/ws", func(c *gin.Context) {
		h.HandleConnection(c)
	})
}
