package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ParticipantStatus models the lifecycle of a participant code inside a project.
type ParticipantStatus string

const (
	ParticipantStatusPending   ParticipantStatus = "pending"
	ParticipantStatusActive    ParticipantStatus = "active"
	ParticipantStatusCompleted ParticipantStatus = "completed"
	ParticipantStatusExpired   ParticipantStatus = "expired"
)

// Participant represents a code-based interviewee within a project.
type Participant struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	Code               string
	DisplayName        string
	ExternalIdentifier string
	Status             ParticipantStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastAccessAt       *time.Time
}

// ParticipantRepository persists interview participant records.
type ParticipantRepository interface {
	Create(ctx context.Context, projectID uuid.UUID, code, displayName, externalIdentifier string) (*Participant, error)
	GetByID(ctx context.Context, participantID uuid.UUID) (*Participant, error)
	GetByCode(ctx context.Context, projectID uuid.UUID, code string) (*Participant, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Participant, error)
	UpdateStatus(ctx context.Context, participantID uuid.UUID, status ParticipantStatus) (*Participant, error)
}

// ParticipantUsecase holds the project business logic for participant codes.
type ParticipantUsecase interface {
	CreateParticipant(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, displayName, externalIdentifier string) (*Participant, error)
	ListParticipants(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) ([]Participant, error)
	GetParticipantByCode(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, code string) (*Participant, error)
	UpdateParticipantStatus(ctx context.Context, actorID uuid.UUID, participantID uuid.UUID, status ParticipantStatus) (*Participant, error)
}
