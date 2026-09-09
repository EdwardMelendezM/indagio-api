package ws

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// RateLimiter implements a sliding-window per-user rate limit, used by
// both the legacy /ws/chat router and the chat-mvp /api/messages/ws
// router. Exported so the chat-mvp router (in a sibling package) can
// share the same in-process limiter.
type RateLimiter struct {
	mu           sync.RWMutex
	windowSize   time.Duration
	maxMessages  int
	messageCount map[uuid.UUID][]time.Time // userID -> timestamps of messages
}

// NewRateLimiter creates a rate limiter.
//
// windowSize:  sliding window length (e.g. 1 second).
// maxMessages: maximum allowed per user per window.
func NewRateLimiter(windowSize time.Duration, maxMessages int) *RateLimiter {
	return &RateLimiter{
		windowSize:   windowSize,
		maxMessages:  maxMessages,
		messageCount: make(map[uuid.UUID][]time.Time),
	}
}

// Allow checks if a user is allowed to send a message.
// Returns true if the user can send, false if rate limited.
func (r *RateLimiter) Allow(userID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.windowSize)

	// Get existing timestamps
	timestamps := r.messageCount[userID]

	// Filter to only keep timestamps within the window
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if user has exceeded the limit
	if len(validTimestamps) >= r.maxMessages {
		r.messageCount[userID] = validTimestamps
		return false
	}

	// Add new timestamp
	validTimestamps = append(validTimestamps, now)
	r.messageCount[userID] = validTimestamps

	return true
}

// Cleanup removes old entries to prevent memory leaks.
// Should be called periodically.
func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-r.windowSize)

	for userID, timestamps := range r.messageCount {
		var validTimestamps []time.Time
		for _, ts := range timestamps {
			if ts.After(windowStart) {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		if len(validTimestamps) == 0 {
			delete(r.messageCount, userID)
		} else {
			r.messageCount[userID] = validTimestamps
		}
	}
}
