package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

// Default rate limiting settings
const (
	DefaultRateLimitWindow = 1 * time.Second
	DefaultRateLimitMax    = 10 // messages per window
)

// Router dispatches WebSocket events to use cases
type Router struct {
	hub         *Hub
	messageUC   domain.MessageUsecase
	convRepo    domain.ConversationRepository
	logger      *slog.Logger
	rateLimiter *RateLimiter
}

// NewRouter creates a new WebSocket router. The rate limiter is built
// internally with DefaultRateLimitWindow / DefaultRateLimitMax. Callers
// that want to share a single limiter across legacy + chat-mvp routers
// can construct one with NewRateLimiter and inject via SetRateLimiter.
func NewRouter(
	hub *Hub,
	messageUC domain.MessageUsecase,
	convRepo domain.ConversationRepository,
	logger *slog.Logger,
) *Router {
	return &Router{
		hub:         hub,
		messageUC:   messageUC,
		convRepo:    convRepo,
		logger:      logger,
		rateLimiter: NewRateLimiter(DefaultRateLimitWindow, DefaultRateLimitMax),
	}
}

// SetRateLimiter swaps in an externally-managed RateLimiter (e.g. one
// shared with the chat-mvp router). When nil the router falls back to
// an internal no-op limiter — see EnsureRateLimiter.
func (r *Router) SetRateLimiter(rl *RateLimiter) {
	r.rateLimiter = rl
}

// EnsureRateLimiter makes sure r.rateLimiter is non-nil. Called by
// dispatch paths so a forgotten SetRateLimiter can't cause nil-deref.
func (r *Router) EnsureRateLimiter() {
	if r.rateLimiter == nil {
		r.rateLimiter = NewRateLimiter(DefaultRateLimitWindow, DefaultRateLimitMax)
	}
}

// Dispatch routes an incoming envelope to the appropriate handler
func (r *Router) Dispatch(c *Client, env domain.WSEnvelope) {
	ctx := context.Background()

	switch env.Type {
	case domain.WSEventSendMessage:
		r.handleSendMessage(ctx, c, env)

	case domain.WSEventMarkRead:
		r.handleMarkRead(ctx, c, env)

	case domain.WSEventTyping:
		r.handleTyping(ctx, c, env)

	case domain.WSEventPing:
		c.Send(domain.WSEnvelope{Type: domain.WSEventPong})

	case domain.WSEventEditMessage:
		r.handleEditMessage(ctx, c, env)

	case domain.WSEventDeleteMessage:
		r.handleDeleteMessage(ctx, c, env)

	case domain.WSEventActiveConversation:
		r.handleActiveConversation(ctx, c, env)

	default:
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
	}
}

func (r *Router) handleSendMessage(ctx context.Context, c *Client, env domain.WSEnvelope) {
	// Check rate limit
	r.EnsureRateLimiter()
	if !r.rateLimiter.Allow(c.UserID) {
		c.Send(errorEnvelope(env.RequestID, domain.ErrRateLimited))
		return
	}

	var payload SendMessagePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	conversationID, err := uuid.Parse(payload.ConversationID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	var replyToID *uuid.UUID
	if payload.ReplyToID != "" {
		parsed, err := uuid.Parse(payload.ReplyToID)
		if err == nil {
			replyToID = &parsed
		}
	}

	msg, err := r.messageUC.SendMessageWithMedia(ctx, conversationID, c.UserID, domain.SendMessageInput{
		Body:      payload.Body,
		ReplyToID: replyToID,
		MediaURLs: payload.MediaURLs,
	})
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}

	// Send ack to the sender with the full message payload — same shape as
	// the broadcast that the other participants will receive. Going through
	// `BuildBroadcastPayload` keeps the ack and the broadcast in lockstep.
	ack, err := r.messageUC.BuildBroadcastPayload(ctx, msg)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}
	c.Send(ackEnvelope(env.RequestID, ack))
}

func (r *Router) handleActiveConversation(ctx context.Context, c *Client, env domain.WSEnvelope) {
	var payload struct {
		ConversationID *string `json:"conversation_id"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		// Low-priority hint frame: log and continue, don't close connection
		r.logger.Debug("active_conversation: unmarshal failed", "err", err)
		return
	}

	var convID *uuid.UUID
	if payload.ConversationID != nil {
		parsed, err := uuid.Parse(*payload.ConversationID)
		if err != nil {
			r.logger.Debug("active_conversation: invalid uuid", "val", *payload.ConversationID)
			return
		}
		convID = &parsed
	}

	c.hub.GetPresenceTracker().SetActiveConversation(c.UserID, c.connID, convID)
}

func (r *Router) handleMarkRead(ctx context.Context, c *Client, env domain.WSEnvelope) {
	var payload MarkReadPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	conversationID, err := uuid.Parse(payload.ConversationID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	lastReadMsgID, err := uuid.Parse(payload.LastReadMessageID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	if err := r.messageUC.MarkAsRead(ctx, conversationID, c.UserID, lastReadMsgID); err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}

	// Broadcast read receipt to other participants
	receiptPayload := ReadReceiptPayload{
		ConversationID:    payload.ConversationID,
		UserID:            c.UserID.String(),
		LastReadMessageID: payload.LastReadMessageID,
	}

	event := domain.WSEnvelope{
		Type:    domain.WSEventReadReceipt,
		Payload: mustMarshal(receiptPayload),
	}
	r.hub.BroadcastToConversation(conversationID, event, &c.UserID)
}

func (r *Router) handleTyping(ctx context.Context, c *Client, env domain.WSEnvelope) {
	var payload TypingPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	conversationID, err := uuid.Parse(payload.ConversationID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	// Broadcast typing indicator to other participants
	indicatorPayload := TypingIndicatorPayload{
		ConversationID: payload.ConversationID,
		UserID:         c.UserID.String(),
		IsTyping:       payload.IsTyping,
	}

	event := domain.WSEnvelope{
		Type:    domain.WSEventTypingIndicator,
		Payload: mustMarshal(indicatorPayload),
	}
	r.hub.BroadcastToConversation(conversationID, event, &c.UserID)
}

func (r *Router) handleEditMessage(ctx context.Context, c *Client, env domain.WSEnvelope) {
	var payload EditMessagePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	_, err = r.messageUC.EditMessage(ctx, messageID, c.UserID, payload.NewBody)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}

	// Broadcasting is handled by the usecase
}

func (r *Router) handleDeleteMessage(ctx context.Context, c *Client, env domain.WSEnvelope) {
	var payload DeleteMessagePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	messageID, err := uuid.Parse(payload.MessageID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, domain.ErrValidation))
		return
	}

	// Get message to find conversation ID before deleting
	_, err = r.messageUC.GetMessage(ctx, messageID)
	if err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}

	if err := r.messageUC.DeleteMessage(ctx, messageID, c.UserID); err != nil {
		c.Send(errorEnvelope(env.RequestID, err))
		return
	}

	// Broadcasting is handled by the usecase
}

// Payload types

type SendMessagePayload struct {
	ConversationID string   `json:"conversation_id"`
	Body           string   `json:"body"`
	ReplyToID      string   `json:"reply_to_id,omitempty"`
	MediaURLs      []string `json:"media_urls,omitempty"`
}

type MarkReadPayload struct {
	ConversationID    string `json:"conversation_id"`
	LastReadMessageID string `json:"last_read_message_id"`
}

type TypingPayload struct {
	ConversationID string `json:"conversation_id"`
	IsTyping       bool   `json:"is_typing"`
}

type EditMessagePayload struct {
	MessageID string `json:"message_id"`
	NewBody   string `json:"new_body"`
}

type DeleteMessagePayload struct {
	MessageID string `json:"message_id"`
}

type ReadReceiptPayload struct {
	ConversationID    string `json:"conversation_id"`
	UserID            string `json:"user_id"`
	LastReadMessageID string `json:"last_read_message_id"`
}

type TypingIndicatorPayload struct {
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
	IsTyping       bool   `json:"is_typing"`
}

// Helper types for message serialization

func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
