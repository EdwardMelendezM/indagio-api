package ws

import (
	"sync"

	"github.com/google/uuid"
)

// PresenceTracker is an in-memory tracker for real-time online status
// It complements the persistent is_online column in the database
type PresenceTracker struct {
	mu     sync.RWMutex
	online map[uuid.UUID]bool

	// connStates tracks the active conversation per connection
	// userID -> connID -> activeConversationID (nil means no active conversation)
	connStates map[uuid.UUID]map[string]*uuid.UUID
}

// NewPresenceTracker creates a new in-memory presence tracker
func NewPresenceTracker() *PresenceTracker {
	return &PresenceTracker{
		online:     make(map[uuid.UUID]bool),
		connStates: make(map[uuid.UUID]map[string]*uuid.UUID),
	}
}

// SetOnline marks a user as online
func (p *PresenceTracker) SetOnline(userID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.online[userID] = true
}

// SetOffline marks a user as offline
func (p *PresenceTracker) SetOffline(userID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.online, userID)
}

// IsOnline returns true if the user is currently connected via WebSocket
// Implements domain.PresenceTracker interface
func (p *PresenceTracker) IsOnline(userID uuid.UUID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.online[userID]
}

// OnlineUserIDs filters a list of user IDs to only include those currently online
// Implements domain.PresenceTracker interface
func (p *PresenceTracker) OnlineUserIDs(userIDs []uuid.UUID) []uuid.UUID {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var online []uuid.UUID
	for _, id := range userIDs {
		if p.online[id] {
			online = append(online, id)
		}
	}
	return online
}

// SetActiveConversation updates the active conversation for a specific connection.
// conversationID nil means "no active conversation" (listing, settings, background)
func (p *PresenceTracker) SetActiveConversation(userID uuid.UUID, connID string, conversationID *uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connStates[userID] == nil {
		p.connStates[userID] = make(map[string]*uuid.UUID)
	}
	p.connStates[userID][connID] = conversationID
}

// IsActiveInConversation returns true if the user has any connection actively
// viewing the specified conversation. Used to gate push notifications.
func (p *PresenceTracker) IsActiveInConversation(userID, conversationID uuid.UUID) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conns, ok := p.connStates[userID]
	if !ok {
		return false
	}
	for _, activeConv := range conns {
		if activeConv != nil && *activeConv == conversationID {
			return true
		}
	}
	return false
}

// RemoveConnection removes all presence state for a specific connection.
// Called when a client disconnects.
func (p *PresenceTracker) RemoveConnection(userID uuid.UUID, connID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conns, ok := p.connStates[userID]; ok {
		delete(conns, connID)
		if len(conns) == 0 {
			delete(p.connStates, userID)
		}
	}
}
