package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

// WSEventType defines the types of WebSocket events in the chat protocol
type WSEventType string

const (
	// Client -> Server events (legacy /api/ws/chat protocol)
	WSEventSendMessage        WSEventType = "send_message"
	WSEventEditMessage        WSEventType = "edit_message"
	WSEventDeleteMessage      WSEventType = "delete_message"
	WSEventMarkRead           WSEventType = "mark_read"
	WSEventTyping             WSEventType = "typing"
	WSEventPing               WSEventType = "ping"
	WSEventActiveConversation WSEventType = "active_conversation" // frontend reports which conversation is active

	// Server -> Client events (legacy /api/ws/chat protocol)
	WSEventMessageNew      WSEventType = "message_new"
	WSEventMessageEdited   WSEventType = "message_edited"
	WSEventMessageDeleted  WSEventType = "message_deleted"
	WSEventReadReceipt     WSEventType = "read_receipt"
	WSEventTypingIndicator WSEventType = "typing_indicator"
	WSEventPresenceUpdate  WSEventType = "presence_update"
	WSEventError           WSEventType = "error"
	WSEventPong            WSEventType = "pong"

	// ──────────────────────────────────────────────────────────────────────
	// chat-mvp event types (/api/messages/ws). Frozen by docs/47 §2.1.
	// Distinct from the legacy protocol — kept separate so a future
	// deprecation of the legacy surface can delete the legacy types
	// without touching the chat-mvp surface.
	// ──────────────────────────────────────────────────────────────────────

	// Client → Server (chat-mvp)
	WSEventSend WSEventType = "send"
	WSEventRead WSEventType = "read" // chat-mvp mark-as-read; intentionally distinct from WSEventMarkRead (legacy)

	// Server → Client (chat-mvp)
	WSEventMessageAck WSEventType = "ack" // separate frame acknowledging a send, carrying {client_msg_id, server_id, ts}
)

// WSEnvelope is the uniform format for all WebSocket messages
type WSEnvelope struct {
	Type      WSEventType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`              // JSON raw message
	RequestID string          `json:"request_id,omitempty"` // for client to correlate ack
}

// Broadcaster defines the interface for broadcasting events to conversation participants
type Broadcaster interface {
	BroadcastToConversation(conversationID uuid.UUID, event WSEnvelope, excludeUserID *uuid.UUID)
	BroadcastToUser(userID uuid.UUID, event WSEnvelope)
}

// ConnectionCounter defines the interface for counting active WebSocket connections.
type ConnectionCounter interface {
	ActiveConnectionCount() int
}
