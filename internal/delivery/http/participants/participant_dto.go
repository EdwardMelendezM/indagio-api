package participants

import (
	"time"

	"indagio-api/internal/domain"
)

type CreateParticipantRequest struct {
	DisplayName        string `json:"display_name" binding:"required,min=2,max=160"`
	ExternalIdentifier string `json:"external_identifier"`
}

type UpdateParticipantStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type ParticipantResponse struct {
	ID                 string    `json:"id"`
	ProjectID          string    `json:"project_id"`
	Code               string    `json:"code"`
	DisplayName        string    `json:"display_name"`
	ExternalIdentifier string    `json:"external_identifier,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func ToParticipantResponse(p *domain.Participant) ParticipantResponse {
	return ParticipantResponse{
		ID:                 p.ID.String(),
		ProjectID:          p.ProjectID.String(),
		Code:               p.Code,
		DisplayName:        p.DisplayName,
		ExternalIdentifier: p.ExternalIdentifier,
		Status:             string(p.Status),
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
