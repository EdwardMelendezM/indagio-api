package user

import (
	"time"

	"indagio-api/internal/delivery/http/auth"
	"indagio-api/internal/domain"
)

// UpdateUserNameRequest represents the request body for updating username.
// @Description	New name for the user (2-100 characters)
// @Example		{"name": "John Doe"}
type UpdateUserNameRequest struct {
	Name string `json:"name" binding:"required,min=2,max=100"`
}

// UpdateUserNameResponse represents the response for successful name update.
// @Description	Confirmation message
// @Example		{"message": "name updated"}
type UpdateUserNameResponse struct {
	Message string `json:"message"`
}

// ListAvailableUsersQuery holds query parameters for GET /api/users/available.
// @Description	Paginated, searchable list of users the caller can start a
// @Description	conversation with. Caller is always excluded.
type ListAvailableUsersQuery struct {
	Page  int    `form:"page,default=1"  binding:"min=1"`
	Limit int    `form:"limit,default=20" binding:"min=1,max=100"`
	Q     string `form:"q"               binding:"max=80"`
}

// ToUserResponseList converts a slice of domain.User into the public
// auth.UserResponse wire format. Re-using the same DTO as
// ConversationResponse.OtherUser keeps the chat and contact-picker UIs
// consistent — a user shown in the "new chat" list renders the same as
// the other side of an existing conversation.
func ToUserResponseList(users []domain.User) []auth.UserResponse {
	out := make([]auth.UserResponse, 0, len(users))
	for i := range users {
		out = append(out, auth.ToUserResponse(&users[i]))
	}
	return out
}

// PublicUserResponse is the safe-to-expose shape for GET /api/users/:id.
//
// It mirrors auth.UserResponse but drops two fields that must NEVER be
// surfaced for another user:
//
//   - email — PII, only the owner should ever see it (use /api/auth/me).
//   - blocked — internal moderation state, leaking it would tell an
//     attacker which accounts are sanctioned.
//
// Kept: id, name, role, avatar_url, border, created_at. The shape is
// stable enough that the FE can render a public profile page from this
// single response without having to call /auth/me or piece fields
// together from per-thread author objects.
type PublicUserResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Role      string          `json:"role"`
	AvatarURL *string         `json:"avatar_url"`
	Border    *auth.BorderDTO `json:"border,omitempty"`
	CreatedAt string          `json:"created_at"`
}

// ToPublicUserResponse maps a domain.User to the safe public DTO.
func ToPublicUserResponse(u *domain.User) PublicUserResponse {
	resp := PublicUserResponse{
		ID:        u.ID.String(),
		Name:      u.Name,
		Role:      string(u.Role),
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
	if u.SelectedBorder != nil {
		b := u.SelectedBorder
		resp.Border = &auth.BorderDTO{
			ID:           b.ID.String(),
			Slug:         b.Slug,
			Name:         b.Name,
			AssetURL:     b.AssetURL,
			ThumbnailURL: b.ThumbnailURL,
			Tier:         b.Tier,
		}
	}
	return resp
}
