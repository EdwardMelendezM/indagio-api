package chat_mvp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"foro-unsaac-backend/internal/delivery/ws"
	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mocks
// ─────────────────────────────────────────────────────────────────────────────

// fakeMessageUC is a hand-rolled stub of domain.MessageUsecase. It only
// implements the methods the chat-mvp router calls; everything else
// returns zero values.
type fakeMessageUC struct {
	mu sync.Mutex

	// SendMessageWithMediaAndIdempotency result
	sendMsg      *domain.Message
	sendIsNew    bool
	sendErr      error
	sendCalled   int
	lastSendIn   domain.SendMessageInput
	lastConvID   uuid.UUID
	lastSenderID uuid.UUID

	// MarkAsRead
	markReadErr    error
	markReadCalled int
	lastReadMsgID  uuid.UUID

	// BuildBroadcastPayload
	buildPayload domain.MessageBroadcastPayload
	buildErr     error
}

func (f *fakeMessageUC) SendMessageWithMediaAndIdempotency(
	_ context.Context, convID, senderID uuid.UUID, in domain.SendMessageInput,
) (*domain.Message, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalled++
	f.lastConvID = convID
	f.lastSenderID = senderID
	f.lastSendIn = in
	return f.sendMsg, f.sendIsNew, f.sendErr
}

func (f *fakeMessageUC) MarkAsRead(_ context.Context, _ uuid.UUID, _ uuid.UUID, msgID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markReadCalled++
	f.lastReadMsgID = msgID
	return f.markReadErr
}

func (f *fakeMessageUC) BuildBroadcastPayload(_ context.Context, _ *domain.Message) (domain.MessageBroadcastPayload, error) {
	return f.buildPayload, f.buildErr
}

// Stub the rest of the MessageUsecase interface (not called by the chat-mvp
// router but required to satisfy the interface signature).
func (f *fakeMessageUC) GetMessage(_ context.Context, _ uuid.UUID) (*domain.Message, error) {
	return nil, nil
}
func (f *fakeMessageUC) SendMessage(_ context.Context, _, _ uuid.UUID, _ string, _ *uuid.UUID) (*domain.Message, error) {
	return nil, nil
}
func (f *fakeMessageUC) SendMessageWithMedia(_ context.Context, _, _ uuid.UUID, _ domain.SendMessageInput) (*domain.Message, error) {
	return nil, nil
}
func (f *fakeMessageUC) EditMessage(_ context.Context, _, _ uuid.UUID, _ string) (*domain.Message, error) {
	return nil, nil
}
func (f *fakeMessageUC) DeleteMessage(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (f *fakeMessageUC) ListMessages(_ context.Context, _, _ uuid.UUID, _ int, _, _ *uuid.UUID) ([]domain.Message, error) {
	return nil, nil
}
func (f *fakeMessageUC) ListMessagesWithPreviews(_ context.Context, _, _ uuid.UUID, _ int, _, _ *uuid.UUID) ([]domain.Message, map[uuid.UUID]domain.Message, error) {
	return nil, nil, nil
}
func (f *fakeMessageUC) ListMessagesWithHasMore(_ context.Context, _, _ uuid.UUID, _ int, _ *uuid.UUID) ([]domain.Message, bool, error) {
	return nil, false, nil
}

// fakeConvRepo is a hand-rolled stub of domain.ConversationRepository.
type fakeConvRepo struct {
	mu sync.Mutex

	// IsParticipant
	isParticipant    bool
	isParticipantErr error
	isPartCalled     int

	// GetByID
	getByIDResult *domain.Conversation
	getByIDErr    error
	getByIDCalled int
}

func (f *fakeConvRepo) IsParticipant(_ context.Context, _, _ uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.isPartCalled++
	return f.isParticipant, f.isParticipantErr
}

func (f *fakeConvRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getByIDCalled++
	return f.getByIDResult, f.getByIDErr
}

// Stub the rest.
func (f *fakeConvRepo) Create(_ context.Context, _ *domain.Conversation, _ []uuid.UUID) (*domain.Conversation, error) {
	return nil, nil
}
func (f *fakeConvRepo) FindDirectBetween(_ context.Context, _, _ uuid.UUID) (*domain.Conversation, error) {
	return nil, nil
}
func (f *fakeConvRepo) ListForUser(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.ConversationWithMeta, int, error) {
	return nil, 0, nil
}
func (f *fakeConvRepo) GetParticipants(_ context.Context, _ uuid.UUID) ([]domain.ConversationParticipant, error) {
	return nil, nil
}
func (f *fakeConvRepo) UpdateLastMessage(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (f *fakeConvRepo) UpdateLastRead(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}
func (f *fakeConvRepo) GetActiveConversationIDsForUser(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for fix #2: classifyError must use errors.Is so wrapped errors
// are correctly classified.
// ─────────────────────────────────────────────────────────────────────────────

func TestClassifyError_WrappedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "direct ErrForbidden",
			err:  domain.ErrForbidden,
			want: "FORBIDDEN",
		},
		{
			name: "wrapped ErrForbidden via fmt.Errorf",
			err:  fmt.Errorf("check participation: %w", domain.ErrForbidden),
			want: "FORBIDDEN",
		},
		{
			name: "double-wrapped ErrValidation",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", domain.ErrValidation)),
			want: "VALIDATION_ERROR",
		},
		{
			name: "wrapped ErrNotFound",
			err:  fmt.Errorf("repo: %w", domain.ErrNotFound),
			want: "NOT_FOUND",
		},
		{
			name: "wrapped ErrRateLimited",
			err:  fmt.Errorf("rl: %w", domain.ErrRateLimited),
			want: "RATE_LIMITED",
		},
		{
			name: "unrelated error",
			err:  errors.New("boom"),
			want: "INTERNAL_ERROR",
		},
		{
			name: "nil error",
			err:  nil,
			want: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyError(tt.err); got != tt.want {
				t.Errorf("classifyError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for fix #3: handleSend must enforce body length 1..MaxBodyChars.
// ─────────────────────────────────────────────────────────────────────────────

// makeRouter builds a router wired to the supplied fakes. The hub is
// nil — handleSend doesn't touch it; only handleRead does, and that
// path is covered separately.
func makeRouter(msgUC *fakeMessageUC, convRepo *fakeConvRepo, rl *ws.RateLimiter) *Router {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	return NewRouter(nil, msgUC, convRepo, nil, rl, logger)
}

type testWriter struct{}

func (w testWriter) Write(p []byte) (int, error) { return len(p), nil }

// newTestClient returns a Client whose send channel is drained by
// reading from c.SendChannel(). The websocket.Conn is nil because we
// never call readPump / writePump in unit tests.
func newTestClient(userID uuid.UUID) *ws.Client {
	return ws.NewClient(userID, nil, nil, slog.New(slog.NewTextHandler(testWriter{}, nil)))
}

func TestHandleSend_BodyLengthCap(t *testing.T) {
	msgUC := &fakeMessageUC{
		sendMsg:   &domain.Message{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		sendIsNew: true,
	}
	convRepo := &fakeConvRepo{}
	rl := ws.NewRateLimiter(time.Second, 100) // generous so it doesn't trip
	r := makeRouter(msgUC, convRepo, rl)

	c := newTestClient(uuid.New())

	// Body longer than MaxBodyChars → 422-shaped error frame, send NOT called.
	bigBody := make([]byte, MaxBodyChars+1)
	for i := range bigBody {
		bigBody[i] = 'a'
	}

	payload, _ := json.Marshal(SendPayload{
		ClientMsgID:    "cmid-1",
		ConversationID: uuid.New().String(),
		Body:           string(bigBody),
	})
	r.handleSend(context.Background(), c, domain.WSEnvelope{
		Type:      domain.WSEventSend,
		Payload:   payload,
		RequestID: "req-1",
	})

	if msgUC.sendCalled != 0 {
		t.Errorf("send should not have been called for over-length body; got %d calls", msgUC.sendCalled)
	}

	// Drain the send channel and verify an error frame went out.
	select {
	case env := <-c.SendChannel():
		var got ErrorPayload
		if err := json.Unmarshal(env.Payload, &got); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if got.Code != "VALIDATION_ERROR" {
			t.Errorf("expected VALIDATION_ERROR code, got %q (msg=%q)", got.Code, got.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected an error frame on the send channel; got nothing")
	}
}

func TestHandleSend_EmptyBodyRejected(t *testing.T) {
	msgUC := &fakeMessageUC{}
	convRepo := &fakeConvRepo{}
	rl := ws.NewRateLimiter(time.Second, 100)
	r := makeRouter(msgUC, convRepo, rl)

	c := newTestClient(uuid.New())
	payload, _ := json.Marshal(SendPayload{
		ClientMsgID:    "cmid-1",
		ConversationID: uuid.New().String(),
		Body:           "",
	})
	r.handleSend(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventSend,
		Payload: payload,
	})

	if msgUC.sendCalled != 0 {
		t.Errorf("send should not have been called for empty body")
	}
	select {
	case env := <-c.SendChannel():
		var got ErrorPayload
		_ = json.Unmarshal(env.Payload, &got)
		if got.Code != "VALIDATION_ERROR" {
			t.Errorf("expected VALIDATION_ERROR; got %q", got.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected an error frame")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for fix #4: handleRead must reject non-participants with FORBIDDEN.
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleRead_NonParticipantRejected(t *testing.T) {
	msgUC := &fakeMessageUC{}
	convRepo := &fakeConvRepo{
		isParticipant: false, // the call
		getByIDResult: &domain.Conversation{ID: uuid.New(), LastMessageID: nil},
	}
	r := makeRouter(msgUC, convRepo, nil)

	c := newTestClient(uuid.New())
	convID := uuid.New()
	payload, _ := json.Marshal(ReadPayload{ConversationID: convID.String()})

	r.handleRead(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventRead,
		Payload: payload,
	})

	if convRepo.isPartCalled != 1 {
		t.Errorf("IsParticipant should have been called once; got %d", convRepo.isPartCalled)
	}
	if convRepo.getByIDCalled != 0 {
		t.Errorf("GetByID must NOT be called for non-participants; got %d calls", convRepo.getByIDCalled)
	}
	if msgUC.markReadCalled != 0 {
		t.Errorf("MarkAsRead must NOT be called for non-participants; got %d calls", msgUC.markReadCalled)
	}
	select {
	case env := <-c.SendChannel():
		var got ErrorPayload
		_ = json.Unmarshal(env.Payload, &got)
		if got.Code != "FORBIDDEN" {
			t.Errorf("expected FORBIDDEN code; got %q (msg=%q)", got.Code, got.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected an error frame on non-participant read")
	}
}

func TestHandleRead_Participant_NoMessages(t *testing.T) {
	msgUC := &fakeMessageUC{}
	convID := uuid.New()
	convRepo := &fakeConvRepo{
		isParticipant: true,
		getByIDResult: &domain.Conversation{ID: convID, LastMessageID: nil},
	}
	r := makeRouter(msgUC, convRepo, nil)

	c := newTestClient(uuid.New())
	payload, _ := json.Marshal(ReadPayload{ConversationID: convID.String()})

	r.handleRead(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventRead,
		Payload: payload,
	})

	if msgUC.markReadCalled != 0 {
		t.Errorf("MarkAsRead must not run when conversation has no messages; got %d calls", msgUC.markReadCalled)
	}
	select {
	case env := <-c.SendChannel():
		if env.Type != domain.WSEventRead {
			t.Errorf("expected WSEventRead frame; got %q", env.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected a read receipt for empty-thread read")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for fix #5: rate limiter must cap sends to DefaultRateLimitMax/window.
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleSend_RateLimited(t *testing.T) {
	msgUC := &fakeMessageUC{
		sendMsg:   &domain.Message{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		sendIsNew: true,
	}
	convRepo := &fakeConvRepo{}
	rl := ws.NewRateLimiter(time.Second, 2) // tight cap: 2 sends per second
	r := makeRouter(msgUC, convRepo, rl)

	c := newTestClient(uuid.New())
	convID := uuid.New()
	mkPayload := func() json.RawMessage {
		p, _ := json.Marshal(SendPayload{
			ClientMsgID:    uuid.New().String(),
			ConversationID: convID.String(),
			Body:           "hi",
		})
		return p
	}

	// First two sends pass; third one is rate-limited.
	for i := 0; i < 2; i++ {
		r.handleSend(context.Background(), c, domain.WSEnvelope{
			Type:    domain.WSEventSend,
			Payload: mkPayload(),
		})
	}

	// Drain the acks from the first two sends.
	for i := 0; i < 2; i++ {
		select {
		case <-c.SendChannel():
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("send %d: expected ack frame, got nothing", i+1)
		}
	}

	// Third send must be rate-limited.
	r.handleSend(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventSend,
		Payload: mkPayload(),
	})

	if msgUC.sendCalled != 2 {
		t.Errorf("send called %d times; expected 2 (the third should be RL'd)", msgUC.sendCalled)
	}
	select {
	case env := <-c.SendChannel():
		var got ErrorPayload
		_ = json.Unmarshal(env.Payload, &got)
		if got.Code != "RATE_LIMITED" {
			t.Errorf("expected RATE_LIMITED code; got %q (msg=%q)", got.Code, got.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected RATE_LIMITED error frame")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for fix #2 (router side): handleSend with wrapped errors from the
// usecase must surface a classified error frame, not INTERNAL_ERROR.
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleSend_ClassifiesWrappedForbiddenError(t *testing.T) {
	// Usecase returns ErrForbidden wrapped — the previous `err ==`
	// bug would have surfaced this as INTERNAL_ERROR. After the fix
	// it should be classified as FORBIDDEN.
	msgUC := &fakeMessageUC{
		sendErr: fmt.Errorf("check participation: %w", domain.ErrForbidden),
	}
	convRepo := &fakeConvRepo{}
	r := makeRouter(msgUC, convRepo, ws.NewRateLimiter(time.Second, 100))

	c := newTestClient(uuid.New())
	convID := uuid.New()
	payload, _ := json.Marshal(SendPayload{
		ClientMsgID:    "cmid-x",
		ConversationID: convID.String(),
		Body:           "hi",
	})
	r.handleSend(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventSend,
		Payload: payload,
	})

	select {
	case env := <-c.SendChannel():
		var got ErrorPayload
		_ = json.Unmarshal(env.Payload, &got)
		if got.Code != "FORBIDDEN" {
			t.Errorf("expected FORBIDDEN for wrapped ErrForbidden; got %q", got.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected an error frame")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for docs/52 fix: chat-mvp router must set BroadcastToSender=true
// on the SendMessageInput so the usecase broadcasts message_new to ALL
// participants (including the sender). Without this flag the chat-mvp
// FE never receives the broadcast it's listening for.
// ─────────────────────────────────────────────────────────────────────────────

func TestHandleSend_SetsBroadcastToSender(t *testing.T) {
	msgUC := &fakeMessageUC{
		sendMsg:   &domain.Message{ID: uuid.New(), CreatedAt: time.Now().UTC()},
		sendIsNew: true,
	}
	convRepo := &fakeConvRepo{}
	r := makeRouter(msgUC, convRepo, ws.NewRateLimiter(time.Second, 100))

	c := newTestClient(uuid.New())
	convID := uuid.New()
	payload, _ := json.Marshal(SendPayload{
		ClientMsgID:    "cmid-broadcast",
		ConversationID: convID.String(),
		Body:           "hi",
	})
	r.handleSend(context.Background(), c, domain.WSEnvelope{
		Type:    domain.WSEventSend,
		Payload: payload,
	})

	if msgUC.sendCalled != 1 {
		t.Fatalf("expected 1 send call; got %d", msgUC.sendCalled)
	}
	if !msgUC.lastSendIn.BroadcastToSender {
		t.Errorf("chat-mvp handleSend MUST set BroadcastToSender=true; got false. " +
			"This is the docs/52 regression — the sender's connection would never " +
			"receive the message_new frame the FE relies on for the optimistic-send state machine.")
	}

	// Drain the ack frame so the test doesn't leak a goroutine.
	select {
	case <-c.SendChannel():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected an ack frame")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests for docs/55 §2 (B.1) — chat-mvp router handles
// `active_conversation` so the BE can suppress push while the user is
// reading. Without this handler the chat-mvp FE never sets the
// presence-tracker's active_conv and the user gets push notifications
// for messages they're already viewing.
// ─────────────────────────────────────────────────────────────────────────────

// fakePresenceTracker captures the calls the chat-mvp router makes.
// Tests inspect `calls` after invoking Dispatch.
type fakePresenceTracker struct {
	mu    sync.Mutex
	calls []activeConvCall
}

type activeConvCall struct {
	UserID uuid.UUID
	ConnID string
	ConvID *uuid.UUID
}

func (f *fakePresenceTracker) IsOnline(uuid.UUID) bool               { return false }
func (f *fakePresenceTracker) OnlineUserIDs([]uuid.UUID) []uuid.UUID { return nil }
func (f *fakePresenceTracker) SetActiveConversation(u uuid.UUID, c string, conv *uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, activeConvCall{UserID: u, ConnID: c, ConvID: conv})
}
func (f *fakePresenceTracker) IsActiveInConversation(_, _ uuid.UUID) bool { return false }

func TestDispatch_ActiveConversation_SetsState(t *testing.T) {
	pt := &fakePresenceTracker{}
	r := NewRouter(nil, &fakeMessageUC{}, &fakeConvRepo{}, pt,
		ws.NewRateLimiter(time.Second, 100), nil)

	c := newTestClient(uuid.New())
	convID := uuid.New()
	payload, _ := json.Marshal(ActiveConversationPayload{
		ConversationID: ptrString(convID.String()),
	})
	r.Dispatch(c, domain.WSEnvelope{
		Type:    domain.WSEventActiveConversation,
		Payload: payload,
	})

	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.calls) != 1 {
		t.Fatalf("expected 1 SetActiveConversation call; got %d", len(pt.calls))
	}
	got := pt.calls[0]
	if got.UserID != c.UserID {
		t.Errorf("user_id: got %s, want %s", got.UserID, c.UserID)
	}
	if got.ConnID != c.ConnID() {
		t.Errorf("conn_id: got %s, want %s", got.ConnID, c.ConnID())
	}
	if got.ConvID == nil || *got.ConvID != convID {
		t.Errorf("conv_id: got %v, want %s", got.ConvID, convID)
	}
}

func TestDispatch_ActiveConversation_NullClearsState(t *testing.T) {
	pt := &fakePresenceTracker{}
	r := NewRouter(nil, &fakeMessageUC{}, &fakeConvRepo{}, pt,
		ws.NewRateLimiter(time.Second, 100), nil)

	c := newTestClient(uuid.New())
	payload, _ := json.Marshal(ActiveConversationPayload{
		ConversationID: nil, // explicit clear
	})
	r.Dispatch(c, domain.WSEnvelope{
		Type:    domain.WSEventActiveConversation,
		Payload: payload,
	})

	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.calls) != 1 {
		t.Fatalf("expected 1 call; got %d", len(pt.calls))
	}
	if pt.calls[0].ConvID != nil {
		t.Errorf("nil ConversationID must produce nil convID; got %v", pt.calls[0].ConvID)
	}
}

func TestDispatch_ActiveConversation_InvalidUUIDDropsSilently(t *testing.T) {
	pt := &fakePresenceTracker{}
	r := NewRouter(nil, &fakeMessageUC{}, &fakeConvRepo{}, pt,
		ws.NewRateLimiter(time.Second, 100), nil)

	c := newTestClient(uuid.New())
	payload, _ := json.Marshal(ActiveConversationPayload{
		ConversationID: ptrString("not-a-uuid"),
	})
	r.Dispatch(c, domain.WSEnvelope{
		Type:    domain.WSEventActiveConversation,
		Payload: payload,
	})

	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.calls) != 0 {
		t.Errorf("invalid uuid must be dropped silently; got %d calls", len(pt.calls))
	}

	// Also: NO error frame went out. Drain the channel briefly.
	select {
	case env := <-c.SendChannel():
		t.Errorf("silent drop must NOT emit a frame; got %+v", env)
	case <-time.After(50 * time.Millisecond):
		// expected — nothing was emitted
	}
}

func TestDispatch_ActiveConversation_MalformedJSONDropsSilently(t *testing.T) {
	pt := &fakePresenceTracker{}
	r := NewRouter(nil, &fakeMessageUC{}, &fakeConvRepo{}, pt,
		ws.NewRateLimiter(time.Second, 100), nil)

	c := newTestClient(uuid.New())
	r.Dispatch(c, domain.WSEnvelope{
		Type:    domain.WSEventActiveConversation,
		Payload: []byte("not-json"),
	})

	pt.mu.Lock()
	defer pt.mu.Unlock()
	if len(pt.calls) != 0 {
		t.Errorf("malformed JSON must drop; got %d calls", len(pt.calls))
	}
}

// TestDispatch_ActiveConversation_NilPresenceTrackerDrops verifies the
// "router wired without presenceTracker" path is safe — no panic, no
// error frame, just a silent drop. Test environment that doesn't care
// about presence can use this wiring.
func TestDispatch_ActiveConversation_NilPresenceTrackerDrops(t *testing.T) {
	r := NewRouter(nil, &fakeMessageUC{}, &fakeConvRepo{}, nil,
		ws.NewRateLimiter(time.Second, 100), nil)

	c := newTestClient(uuid.New())
	convID := uuid.New()
	payload, _ := json.Marshal(ActiveConversationPayload{
		ConversationID: ptrString(convID.String()),
	})

	// Must not panic.
	r.Dispatch(c, domain.WSEnvelope{
		Type:    domain.WSEventActiveConversation,
		Payload: payload,
	})
}

func ptrString(s string) *string { return &s }
