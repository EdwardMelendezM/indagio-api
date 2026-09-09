package ws

import (
	"context"
	"log/slog"
	"sync"

	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

// Hub manages all WebSocket connections
type Hub struct {
	mu         sync.RWMutex
	clients    map[uuid.UUID]map[*Client]bool // userID -> set of connections (multi-device)
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage

	// Dependencies
	convRepo        domain.ConversationRepository
	presenceUC      domain.PresenceUsecase
	presenceTracker *PresenceTracker
	logger          *slog.Logger
}

// BroadcastMessage holds a message to broadcast to a conversation
type BroadcastMessage struct {
	ConversationID uuid.UUID
	Event          domain.WSEnvelope
	ExcludeUserID  *uuid.UUID
}

// NewHub creates a new Hub instance
func NewHub(
	convRepo domain.ConversationRepository,
	presenceUC domain.PresenceUsecase,
	presenceTracker *PresenceTracker,
	logger *slog.Logger,
) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		clients:         make(map[uuid.UUID]map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan *BroadcastMessage, 256),
		convRepo:        convRepo,
		presenceUC:      presenceUC,
		presenceTracker: presenceTracker,
		logger:          logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.addClient(c)
			if h.presenceTracker != nil {
				h.presenceTracker.SetOnline(c.UserID)
			}
			if h.presenceUC != nil {
				if err := h.presenceUC.MarkOnline(context.Background(), c.UserID); err != nil {
					h.logger.Error("presence mark online failed",
						"user_id", c.UserID, "error", err)
				}
			}

		case c := <-h.unregister:
			h.removeClient(c)

			// Clean up this connection's active conversation state
			if h.presenceTracker != nil {
				h.presenceTracker.RemoveConnection(c.UserID, c.connID)
			}

			if !h.hasOtherConnections(c.UserID) {
				if h.presenceUC != nil {
					if err := h.presenceUC.MarkOffline(context.Background(), c.UserID); err != nil {
						h.logger.Error("presence mark offline failed",
							"user_id", c.UserID, "error", err)
					}
				}
				if h.presenceTracker != nil {
					h.presenceTracker.SetOffline(c.UserID)
				}
			}

		case msg := <-h.broadcast:
			h.broadcastToConversation(msg)
		}
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*Client]bool)
	}
	h.clients[c.UserID][c] = true
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[c.UserID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.UserID)
		}
	}
}

func (h *Hub) hasOtherConnections(userID uuid.UUID) bool {
	if clients, ok := h.clients[userID]; ok {
		return len(clients) > 0
	}
	return false
}

// BroadcastToConversation sends an event to all participants in a conversation.
// Non-blocking: if the broadcast channel is full, the message is dropped with
// a WARN log. Blocking here would deadlock the hub goroutine, which is the
// same goroutine that runs `Run` — a single slow/disconnected client could
// freeze the entire chat.
// Implements domain.Broadcaster interface
func (h *Hub) BroadcastToConversation(conversationID uuid.UUID, event domain.WSEnvelope, excludeUserID *uuid.UUID) {
	msg := &BroadcastMessage{
		ConversationID: conversationID,
		Event:          event,
		ExcludeUserID:  excludeUserID,
	}
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("broadcast channel full, dropping message",
			"conversation_id", conversationID,
			"type", event.Type,
			"exclude_user_id", excludeUserID)
	}
}

func (h *Hub) broadcastToConversation(msg *BroadcastMessage) {
	participants, err := h.convRepo.GetParticipants(context.Background(), msg.ConversationID)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, p := range participants {
		if msg.ExcludeUserID != nil && p.UserID == *msg.ExcludeUserID {
			continue
		}

		for client := range h.clients[p.UserID] {
			client.Send(msg.Event)
		}
	}
}

// BroadcastToUser sends an event to a specific user
// Implements domain.Broadcaster interface
func (h *Hub) BroadcastToUser(userID uuid.UUID, event domain.WSEnvelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[userID] {
		client.Send(event)
	}
}

// Register returns the register channel for new clients
func (h *Hub) Register() chan *Client {
	return h.register
}

// Unregister returns the unregister channel for disconnected clients
func (h *Hub) Unregister() chan *Client {
	return h.unregister
}

// GetPresenceTracker returns the presence tracker
func (h *Hub) GetPresenceTracker() *PresenceTracker {
	return h.presenceTracker
}

// SetPresenceUsecase injects the presence usecase after the hub has been
// constructed. Needed because the presence usecase depends on the hub as
// its broadcaster (for `presence_update` fan-out), so the two must be
// wired in a cycle. Safe to call once, before `Run` starts.
func (h *Hub) SetPresenceUsecase(uc domain.PresenceUsecase) {
	h.presenceUC = uc
}

// ActiveConnectionCount returns the total number of active WebSocket connections.
// Note: This is per-instance. If running multiple replicas behind a load balancer,
// this only counts connections to this specific instance.
func (h *Hub) ActiveConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}
