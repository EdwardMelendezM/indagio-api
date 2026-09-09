package chat_mvp

// Payload types for the chat-mvp WebSocket protocol (/api/messages/ws).
// The wire shape is frozen by docs/47 §2.1 and validated by the FE's
// zod schemas — any field rename here is a breaking change.

// MaxBodyChars is the server-side cap for the message body. It mirrors
// the REST handler's `binding:"required,min=1,max=2000"` on
// ChatMvpSendMessageRequest.Body so the two transports accept the same
// size range — without this guard the WS path silently accepts bodies
// the REST path would 422.
const MaxBodyChars = 2000

// SendPayload is the body of a client → server `send` frame.
type SendPayload struct {
	ClientMsgID    string `json:"client_msg_id"`
	Body           string `json:"body"`
	ConversationID string `json:"conversation_id"`
}

// ReadPayload is the body of a client → server `read` frame.
type ReadPayload struct {
	ConversationID string `json:"conversation_id"`
}

// AckPayload is the body of the server → client `ack` frame, sent in
// response to every `send`. The FE uses this to match the round-trip
// back to its own optimistic-send row.
//
// When the server has already accepted a send with this client_msg_id
// (a network retry, or a WS+RACE fallback) the ack fires a SECOND time
// with the same server_id — no new message_new is broadcast.
type AckPayload struct {
	ClientMsgID string `json:"client_msg_id"`
	ServerID    string `json:"server_id"`
	TS          int64  `json:"ts"` // server unix-millis at the moment of the ack
}

// ReadReceiptPayload is the body of the server → client `read` frame,
// broadcast to all participants (excluding the reader) when someone
// marks a conversation as read.
type ReadReceiptPayload struct {
	ConversationID string `json:"conversation_id"`
	LastReadAt     string `json:"last_read_at"` // RFC3339 timestamp the server set
}

// PingPayload / PongPayload are explicit null bodies; the FE sends
// `{type: "ping", payload: null}` and the server replies with
// `{type: "pong", payload: null}`.
type PingPayload struct{}
type PongPayload struct{}

// ErrorPayload is the body of the server → client `error` frame.
type ErrorPayload struct {
	Code    string `json:"code"`    // e.g. "VALIDATION_ERROR", "RATE_LIMITED"
	Message string `json:"message"` // human-readable
}

// ActiveConversationPayload is the body of the client → server
// `active_conversation` frame. The FE sends this when the user opens
// (or closes) a chat so the BE can suppress push notifications for
// messages the user is already reading.
//
// Per docs/55 §2.4:
//   - non-nil conversation_id = the chat is currently active; BE
//     suppresses push for this user while it remains active.
//   - nil conversation_id = no chat is active; push should resume.
//
// Sending an invalid uuid OR a conversation the caller isn't a
// participant in is a no-op (drop silently per docs/55 §2.4 — a
// `error` frame is acceptable too, but we prefer silent drop because
// this frame is fire-and-forget UX, not a request).
type ActiveConversationPayload struct {
	ConversationID *string `json:"conversation_id"`
}
