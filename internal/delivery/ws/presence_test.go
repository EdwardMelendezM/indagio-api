package ws

import (
	"testing"

	"github.com/google/uuid"
)

// These tests pin down the semantics of the in-memory tracker that drives
// the chat push-suppression gate (IsActiveInConversation) and the
// presence_update fan-out. They are pure unit tests — no DB, no network.

func ptr(u uuid.UUID) *uuid.UUID { return &u }

func TestIsActiveInConversation_SingleDevice(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()
	other := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", &convA)

	if !tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected IsActiveInConversation(user, convA) == true")
	}
	if tr.IsActiveInConversation(user, other) {
		t.Fatalf("expected IsActiveInConversation(user, other) == false")
	}
}

func TestIsActiveInConversation_NilConv(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", nil)

	if tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected IsActiveInConversation(user, convA) == false when active_conversation is nil")
	}
}

func TestIsActiveInConversation_NoActiveConversation(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()

	tr.SetOnline(user)
	// never set active conversation

	if tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected false when no active conversation was ever set")
	}
}

func TestIsActiveInConversation_MultiDevice_OneInTarget(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()
	convB := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", &convA)
	tr.SetActiveConversation(user, "conn2", &convB)

	// ANY connection of the user in convA → push is suppressed for convA
	if !tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected IsActiveInConversation(user, convA) == true (conn1 is in convA)")
	}
	// And same for convB (any connection matches)
	if !tr.IsActiveInConversation(user, convB) {
		t.Fatalf("expected IsActiveInConversation(user, convB) == true (conn2 is in convB)")
	}
}

func TestSetActiveConversation_Overwrites(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()
	convB := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", &convA)
	tr.SetActiveConversation(user, "conn1", &convB)

	if tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected false for convA after overwrite to convB")
	}
	if !tr.IsActiveInConversation(user, convB) {
		t.Fatalf("expected true for convB after overwrite")
	}
}

func TestRemoveConnection_ClearsState(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()
	convB := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", &convA)
	tr.SetActiveConversation(user, "conn2", &convB)

	tr.RemoveConnection(user, "conn1")

	if tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected false for convA after RemoveConnection(conn1)")
	}
	// conn2's state must remain intact
	if !tr.IsActiveInConversation(user, convB) {
		t.Fatalf("expected true for convB after RemoveConnection(conn1) — conn2 still has it")
	}
}

func TestRemoveConnection_LastConn_RemovesEntry(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()
	convA := uuid.New()

	tr.SetOnline(user)
	tr.SetActiveConversation(user, "conn1", &convA)
	tr.RemoveConnection(user, "conn1")

	// After removing the last connection the user entry should be wiped
	// entirely. IsActiveInConversation must return false.
	if tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected false after last connection removed")
	}

	// Re-using the same connID should not resurrect the old state.
	tr.SetActiveConversation(user, "conn1", &convA)
	if !tr.IsActiveInConversation(user, convA) {
		t.Fatalf("expected true after re-SetActiveConversation with a fresh state")
	}
}

func TestIsOnline_TracksAcrossConnections(t *testing.T) {
	tr := NewPresenceTracker()
	user := uuid.New()

	tr.SetOnline(user)
	if !tr.IsOnline(user) {
		t.Fatalf("expected IsOnline == true after SetOnline")
	}
	tr.SetOffline(user)
	if tr.IsOnline(user) {
		t.Fatalf("expected IsOnline == false after SetOffline")
	}
}

func TestOnlineUserIDs_FiltersCorrectly(t *testing.T) {
	tr := NewPresenceTracker()
	users := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	tr.SetOnline(users[0])
	tr.SetOnline(users[2])

	online := tr.OnlineUserIDs(users)
	if len(online) != 2 {
		t.Fatalf("expected 2 online users, got %d", len(online))
	}
	// Verify which ones
	seen := map[uuid.UUID]bool{}
	for _, u := range online {
		seen[u] = true
	}
	if !seen[users[0]] || !seen[users[2]] {
		t.Fatalf("expected users[0] and users[2] in result, got %v", online)
	}
	if seen[users[1]] {
		t.Fatalf("did not expect users[1] in result")
	}
}

// Helper for the "ptr" function to silence unused import warnings if every
// test is commented out temporarily.
var _ = ptr
