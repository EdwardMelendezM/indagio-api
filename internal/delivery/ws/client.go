package ws

import (
	"encoding/json"
	"log/slog"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// DefaultMaxMessageSize is the legacy /ws/chat frame size cap.
	// Kept low because the legacy surface only carries short commands
	// (no sticker/reply metadata). The chat-mvp surface uses a
	// larger cap (8 KB) and overrides per-client at upgrade time.
	DefaultMaxMessageSize int64 = 512

	// ChatMvpMaxMessageSize is the chat-mvp /api/messages/ws cap.
	// Generous enough for a future sticker+reply payload without
	// making the server vulnerable to large-frame DoS.
	ChatMvpMaxMessageSize int64 = 8 * 1024
)

// Dispatcher is the protocol-agnostic contract every WebSocket router
// satisfies. The legacy *Router (in this package) and the chat-mvp
// *Router (in internal/delivery/ws/chat_mvp) both implement it; the
// choice is made at WS-upgrade time and the matching dispatcher is
// passed to readPump.
type Dispatcher interface {
	Dispatch(c *Client, env domain.WSEnvelope)
}

// Client represents a single WebSocket connection
type Client struct {
	UserID uuid.UUID
	connID string // unique identifier for this connection (used for per-connection presence tracking)
	conn   *websocket.Conn
	send   chan domain.WSEnvelope
	hub    *Hub
	logger *slog.Logger
	// MaxMessageSize overrides the read frame cap for this client. When
	// zero the legacy DefaultMaxMessageSize is used. The chat-mvp
	// upgrader sets ChatMvpMaxMessageSize so the FE can send sticker
	// metadata without hitting a frame-too-large error.
	MaxMessageSize int64
	// Protocol identifies the wire contract this client speaks
	// ("legacy" | "chat-mvp"). The dispatcher branch on this so a
	// future v1.1 protocol doesn't have to grow a separate field.
	Protocol string
}

// NewClient creates a new WebSocket client
func NewClient(userID uuid.UUID, conn *websocket.Conn, hub *Hub, logger *slog.Logger) *Client {
	return &Client{
		UserID:         userID,
		connID:         uuid.New().String(), // unique per connection for per-connection presence tracking
		conn:           conn,
		send:           make(chan domain.WSEnvelope, 256),
		hub:            hub,
		logger:         logger,
		MaxMessageSize: DefaultMaxMessageSize,
		Protocol:       "legacy",
	}
}

// Send sends an envelope to the client
func (c *Client) Send(event domain.WSEnvelope) {
	select {
	case c.send <- event:
	default:
		// Client's send buffer is full, drop the message
		c.logger.Warn("client send buffer full, dropping message", "user_id", c.UserID)
	}
}

// SendChannel exposes the per-client outbound channel as a receive-only
// handle. Used by tests in sibling packages (chat_mvp router tests) to
// drain what a handler sent without exposing the unexported field.
// Production code MUST keep using Send().
func (c *Client) SendChannel() <-chan domain.WSEnvelope {
	return c.send
}

// ConnID returns the per-connection identifier used by the hub for
// presence tracking (e.g. SetActiveConversation uses it to scope the
// "user is in chat X" state to this specific connection). Exposed so
// sibling packages (chat_mvp router) can pass it to the hub without
// breaking encapsulation.
func (c *Client) ConnID() string {
	return c.connID
}

// ReadPump and WritePump are exported wrappers so the chat-mvp handler
// (a different package) can launch the per-connection goroutines. The
// lowercase originals remain the implementation; we keep these
// wrappers thin so the encapsulation stays in one place.
func (c *Client) ReadPump(dispatcher Dispatcher) { c.readPump(dispatcher) }
func (c *Client) WritePump()                     { c.writePump() }

// readPump pumps messages from the websocket connection to the dispatcher.
// dispatcher is protocol-specific (legacy or chat-mvp); the caller
// chooses the right one at upgrade time. Both routers share the same
// domain.WSEnvelope shape so this method stays generic.
func (c *Client) readPump(dispatcher Dispatcher) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	maxSize := c.MaxMessageSize
	if maxSize <= 0 {
		maxSize = DefaultMaxMessageSize
	}
	c.conn.SetReadLimit(maxSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("websocket read error", "error", err)
			}
			break
		}

		var env domain.WSEnvelope
		if err := json.Unmarshal(message, &env); err != nil {
			c.logger.Error("failed to unmarshal ws envelope", "error", err)
			c.Send(errorEnvelope("", domain.ErrValidation))
			continue
		}

		dispatcher.Dispatch(c, env)
	}
}

// writePump pumps messages from the send channel to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case env, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the send channel
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}

			if err := c.conn.WriteJSON(env); err != nil {
				c.logger.Error("failed to write ws message", "error", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Helper to create error envelope
func errorEnvelope(requestID string, err error) domain.WSEnvelope {
	appErr, ok := err.(domain.AppError)
	if !ok {
		appErr = domain.AppError{Code: 500, Message: "internal error"}
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"code":    appErr.Code,
		"message": appErr.Message,
	})

	return domain.WSEnvelope{
		Type:      domain.WSEventError,
		Payload:   payload,
		RequestID: requestID,
	}
}

// Helper to create ack envelope
func ackEnvelope(requestID string, payload interface{}) domain.WSEnvelope {
	payloadBytes, _ := json.Marshal(payload)
	return domain.WSEnvelope{
		Type:      domain.WSEventMessageNew,
		Payload:   payloadBytes,
		RequestID: requestID,
	}
}
