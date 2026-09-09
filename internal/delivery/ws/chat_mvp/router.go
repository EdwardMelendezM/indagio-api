package chat_mvp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"indagio-api/internal/delivery/ws"
	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

// Router is the chat-mvp WebSocket dispatcher. It speaks the wire
// contract from docs/47 §2.1 — distinct frame types (send, ack, read,
// ping) and a separate ack frame per send so the FE's optimistic-send
// state machine has a deterministic round-trip.
//
// All mutation paths call into the existing message usecase; this
// router only owns the wire-shape translation. Idempotency, broadcast,
// push — all stay in the usecase layer where they belong.
type Router struct {
	hub             *ws.Hub
	messageUC       domain.MessageUsecase
	convRepo        domain.ConversationRepository
	presenceTracker domain.PresenceTracker // optional — nil disables active_conversation wiring
	logger          *slog.Logger
	rateLimiter     *ws.RateLimiter // optional — nil falls back to a fresh limiter
}

// NewRouter wires the chat-mvp dispatcher. The same usecases the
// legacy /ws/chat router uses are reused here.
//
// Pass nil for rateLimiter to get a fresh in-process limiter. To
// share the legacy router's limiter (recommended — both surfaces are
// the same user and should share the budget) construct one via
// ws.NewRateLimiter and pass it to both.
//
// Pass nil for presenceTracker to disable the active_conversation
// suppression path (the handler will silently drop such frames). In
// production main.go passes the same PresenceTracker the legacy
// router uses so a user's WS connections share state.
//
// A nil logger is replaced with slog.Default() so the router is safe
// to construct in tests that don't care about logging — Debug / Warn
// calls never panic.
func NewRouter(
	hub *ws.Hub,
	messageUC domain.MessageUsecase,
	convRepo domain.ConversationRepository,
	presenceTracker domain.PresenceTracker,
	rateLimiter *ws.RateLimiter,
	logger *slog.Logger,
) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		hub:             hub,
		messageUC:       messageUC,
		convRepo:        convRepo,
		presenceTracker: presenceTracker,
		rateLimiter:     rateLimiter,
		logger:          logger,
	}
}

// ensureRateLimiter guarantees r.rateLimiter is non-nil. Called by
// dispatch paths so a forgotten injection can't cause nil-deref.
func (r *Router) ensureRateLimiter() {
	if r.rateLimiter == nil {
		r.rateLimiter = ws.NewRateLimiter(ws.DefaultRateLimitWindow, ws.DefaultRateLimitMax)
	}
}

// Compile-time check that *Router implements the ws.Dispatcher contract.
var _ ws.Dispatcher = (*Router)(nil)

// Dispatch routes one inbound envelope to the matching handler.
// Unrecognised types emit an `error` frame with code UNKNOWN_EVENT
// rather than closing the connection — chat-mvp clients may grow new
// event types over time and we want forward-compat.
func (r *Router) Dispatch(c *ws.Client, env domain.WSEnvelope) {
	ctx := context.Background()

	switch env.Type {
	case domain.WSEventSend:
		r.handleSend(ctx, c, env)
	case domain.WSEventRead:
		r.handleRead(ctx, c, env)
	case domain.WSEventActiveConversation:
		// UX hint — same fire-and-forget as the legacy router:
		// unmarshal / parse / set state, never close the connection,
		// never emit an error frame (a bad uuid or a conversation the
		// caller isn't a participant in is silently dropped — see
		// handleActiveConversation).
		r.handleActiveConversation(c, env)
	case domain.WSEventPing:
		c.Send(domain.WSEnvelope{
			Type:    domain.WSEventPong,
			Payload: marshalPayload(PongPayload{}),
		})
	default:
		r.sendError(c, env.RequestID, "UNKNOWN_EVENT",
			fmt.Sprintf("event type %q not handled by chat-mvp", env.Type))
	}
}

// handleActiveConversation records which conversation (if any) the
// caller's connection is currently viewing. The BE uses this to
// suppress push notifications while the user is reading (see
// docs/55 §2). The hub also clears this state on connection drop
// (hub.go: RemoveConnection), so the FE does NOT need to send
// "null" before disconnecting.
//
// Behaviour:
//   - valid uuid → store (user_id, conn_id) → active_conv_id
//   - invalid uuid (parse error) → drop silently
//   - missing field / null → clear any previous value for this
//     connection
//
// We do NOT verify "is the caller a participant?" here — the worst
// case is the user sets active_conversation to a foreign conv, and
// push gets suppressed for messages to that user in THAT conv (but
// the user isn't seeing them anyway, so the suppression is moot).
// Verifying participation would add a DB round-trip per
// mount/unmount, which the legacy router avoids for the same reason.
//
// We never emit an `error` frame here: this is UX signalling, not a
// request. A bad value means the FE is confused; surfacing the error
// would just create noise.
func (r *Router) handleActiveConversation(c *ws.Client, env domain.WSEnvelope) {
	if r.presenceTracker == nil {
		return
	}

	var p ActiveConversationPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		r.logger.Debug("chat-mvp active_conversation: unmarshal failed", "err", err)
		return
	}

	var convID *uuid.UUID
	if p.ConversationID != nil && *p.ConversationID != "" {
		parsed, err := uuid.Parse(*p.ConversationID)
		if err != nil {
			r.logger.Debug("chat-mvp active_conversation: invalid uuid", "val", *p.ConversationID)
			return
		}
		convID = &parsed
	}

	r.presenceTracker.SetActiveConversation(c.UserID, c.ConnID(), convID)
}

// handleSend implements the load-bearing idempotent send path.
//
// Wire flow:
//  1. Parse {client_msg_id, body, conversation_id} from the payload.
//  2. Apply rate limit (sliding window per user, shared with the legacy
//     router when both use the same RateLimiter instance).
//  3. Validate the body length 1–2000 to match the REST handler's
//     `binding:"required,min=1,max=2000"` — keeps the two transports
//     consistent.
//  4. Call SendMessageWithMediaAndIdempotency — the (sender_id,
//     client_msg_id) UNIQUE partial index dedupes retries.
//  5. Always emit an `ack` to the sender with {client_msg_id,
//     server_id, ts}. The broadcast (message_new) only happens when
//     isNew=true; the usecase handles that internally.
func (r *Router) handleSend(ctx context.Context, c *ws.Client, env domain.WSEnvelope) {
	// Rate-limit BEFORE unmarshalling so abusive clients get a 429
	// before we burn cycles parsing. The limiter is per-user (shared
	// across the user's connections + the legacy /ws/chat when wired
	// to the same instance).
	r.ensureRateLimiter()
	if !r.rateLimiter.Allow(c.UserID) {
		r.sendError(c, env.RequestID, "RATE_LIMITED",
			"too many sends, slow down")
		return
	}

	var p SendPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR", "invalid send payload")
		return
	}
	if p.ClientMsgID == "" || p.ConversationID == "" || p.Body == "" {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR",
			"client_msg_id, conversation_id, body are required")
		return
	}

	// Length cap matches the REST handler's `binding:"max=2000"`.
	// Without this the two transports accept different body sizes,
	// which breaks the FE's optimistic-send state machine (a send that
	// succeeds over WS but 422s on REST retry will silently lose the
	// message).
	if len(p.Body) > MaxBodyChars {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR",
			fmt.Sprintf("body must be 1-%d chars", MaxBodyChars))
		return
	}

	convID, err := uuid.Parse(p.ConversationID)
	if err != nil {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR", "conversation_id must be a uuid")
		return
	}

	clientMsgID := p.ClientMsgID
	msg, _, err := r.messageUC.SendMessageWithMediaAndIdempotency(
		ctx, convID, c.UserID, domain.SendMessageInput{
			Body:              p.Body,
			ClientMsgID:       &clientMsgID,
			BroadcastToSender: true, // chat-mvp wire: sender must see its own message_new
		},
	)
	if err != nil {
		r.sendError(c, env.RequestID, classifyError(err), err.Error())
		return
	}

	// Ack with the server-assigned id and the timestamp the server set.
	// The usecase has already broadcast message_new when isNew=true; we
	// don't duplicate that here.
	c.Send(domain.WSEnvelope{
		Type: domain.WSEventMessageAck,
		Payload: marshalPayload(AckPayload{
			ClientMsgID: p.ClientMsgID,
			ServerID:    msg.ID.String(),
			TS:          msg.CreatedAt.UnixMilli(),
		}),
		RequestID: env.RequestID,
	})
}

// handleRead marks the conversation as read for the caller and
// broadcasts a `read` receipt to every other participant so their
// UI can clear the unread badge in real time.
//
// Participant authorization is checked up front: a non-participant
// probing for conversation ids gets a `FORBIDDEN` error frame rather
// than a silent no-op (which previously made MarkAsRead return
// success for non-participants because the underlying UPDATE affected
// 0 rows and returned nil).
func (r *Router) handleRead(ctx context.Context, c *ws.Client, env domain.WSEnvelope) {
	var p ReadPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR", "invalid read payload")
		return
	}
	convID, err := uuid.Parse(p.ConversationID)
	if err != nil {
		r.sendError(c, env.RequestID, "VALIDATION_ERROR", "conversation_id must be a uuid")
		return
	}

	// Participant check FIRST — never let a non-participant mutate
	// (or appear to mutate) the conversation's read state.
	isParticipant, err := r.convRepo.IsParticipant(ctx, convID, c.UserID)
	if err != nil {
		r.sendError(c, env.RequestID, classifyError(err), err.Error())
		return
	}
	if !isParticipant {
		r.sendError(c, env.RequestID, "FORBIDDEN", "not a participant")
		return
	}

	// Look up the conversation's last_message_id to satisfy the
	// MarkAsRead signature (legacy takes an explicit message id).
	conv, err := r.convRepo.GetByID(ctx, convID)
	if err != nil || conv.LastMessageID == nil {
		// Conv gone or no messages yet — benign. Ack with a fresh
		// timestamp so the FE clears its unread counter.
		c.Send(domain.WSEnvelope{
			Type: domain.WSEventRead,
			Payload: marshalPayload(ReadReceiptPayload{
				ConversationID: p.ConversationID,
				LastReadAt:     time.Now().UTC().Format(time.RFC3339),
			}),
			RequestID: env.RequestID,
		})
		return
	}

	if err := r.messageUC.MarkAsRead(ctx, convID, c.UserID, *conv.LastMessageID); err != nil {
		r.sendError(c, env.RequestID, classifyError(err), err.Error())
		return
	}

	// Broadcast read receipt to other participants (exclude self).
	receipt := domain.WSEnvelope{
		Type: domain.WSEventRead,
		Payload: marshalPayload(ReadReceiptPayload{
			ConversationID: p.ConversationID,
			LastReadAt:     time.Now().UTC().Format(time.RFC3339),
		}),
	}
	if r.hub != nil {
		r.hub.BroadcastToConversation(convID, receipt, &c.UserID)
	} else {
		// No hub access — fall back to echo to the sender. The FE
		// will at least see its own read go through; other devices
		// sync on reconnect.
		c.Send(receipt)
	}
}

// sendError emits a chat-mvp error frame with a stable code and a
// human-readable message. The FE renders the message in a banner.
func (r *Router) sendError(c *ws.Client, requestID, code, msg string) {
	c.Send(domain.WSEnvelope{
		Type:      domain.WSEventError,
		Payload:   marshalPayload(ErrorPayload{Code: code, Message: msg}),
		RequestID: requestID,
	})
}

// classifyError maps a domain error to a stable FE-facing code.
// Uses errors.Is so wrapped errors (fmt.Errorf("...: %w", domain.ErrX))
// are correctly identified.
func classifyError(err error) string {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return "FORBIDDEN"
	case errors.Is(err, domain.ErrValidation):
		return "VALIDATION_ERROR"
	case errors.Is(err, domain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domain.ErrRateLimited):
		return "RATE_LIMITED"
	default:
		return "INTERNAL_ERROR"
	}
}

// marshalPayload marshals any value to a json.RawMessage (which is what
// domain.WSEnvelope.Payload expects). A marshal failure here would
// indicate a programming error, not a runtime one — we return an empty
// RawMessage so the frame still ships and the broken payload can be
// caught by tests.
func marshalPayload(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return b
}
