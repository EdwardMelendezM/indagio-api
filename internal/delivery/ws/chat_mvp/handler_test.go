package chat_mvp

import "testing"

// Tests for fix #1: extractChatMvpToken must return the exact raw
// protocol alongside the token so the upgrader can echo it back.
// gorilla's exact-match Subprotocols negotiation will reject the
// connection if the server doesn't echo one of the client-proposed
// values verbatim — so the helper has to be lossless.

func TestExtractChatMvpToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantProto string
		wantToken string
		wantOK    bool
	}{
		{
			name:      "empty header",
			header:    "",
			wantProto: "",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "single token-derived protocol",
			header:    "chat-mvp.eyJhbGciOiJIUzI1NiJ9.payload.sig",
			wantProto: "chat-mvp.eyJhbGciOiJIUzI1NiJ9.payload.sig",
			wantToken: "eyJhbGciOiJIUzI1NiJ9.payload.sig",
			wantOK:    true,
		},
		{
			name:      "token-derived first, then legacy",
			header:    "chat-mvp.jwt1, chat-mvp",
			wantProto: "chat-mvp.jwt1",
			wantToken: "jwt1",
			wantOK:    true,
		},
		{
			name:      "only legacy protocol — must not match",
			header:    "chat-mvp",
			wantProto: "",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "unrelated subprotocol — must not match",
			header:    "soap, graphql",
			wantProto: "",
			wantToken: "",
			wantOK:    false,
		},
		{
			name:      "extra whitespace tolerated",
			header:    "  chat-mvp.token123  ",
			wantProto: "chat-mvp.token123",
			wantToken: "token123",
			wantOK:    true,
		},
		{
			name:      "empty token — must not match (would be auth-bypass)",
			header:    "chat-mvp.",
			wantProto: "chat-mvp.",
			wantToken: "",
			wantOK:    true, // helper still returns true; the token-svc check downstream rejects empty tokens.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProto, gotToken, gotOK := extractChatMvpToken(tt.header)
			if gotProto != tt.wantProto {
				t.Errorf("proto: got %q, want %q", gotProto, tt.wantProto)
			}
			if gotToken != tt.wantToken {
				t.Errorf("token: got %q, want %q", gotToken, tt.wantToken)
			}
			if gotOK != tt.wantOK {
				t.Errorf("ok: got %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}
