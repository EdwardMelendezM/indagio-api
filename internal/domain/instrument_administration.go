package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AdministrationStatus identifies the lifecycle of a single instrument application.
type AdministrationStatus string

const (
	AdministrationStatusInProgress AdministrationStatus = "in_progress"
	AdministrationStatusCompleted  AdministrationStatus = "completed"
	AdministrationStatusAbandoned  AdministrationStatus = "abandoned"
)

// InstrumentAdministration groups the answer records a participant submits
// for one attempt at an instrument, pinned to the catalog version active
// when the attempt started.
type InstrumentAdministration struct {
	ID                uuid.UUID
	ProjectID         uuid.UUID
	ParticipantID     uuid.UUID
	InstrumentID      uuid.UUID
	InstrumentVersion int
	Status            AdministrationStatus
	SyncStatus        SyncStatus // reuses the enum defined in answer.go
	ClientGeneratedID *string
	StartedAt         time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// InstrumentScore is a persisted, computed result for one administration.
// SubscaleKey nil means it's the instrument's total score.
type InstrumentScore struct {
	ID                 uuid.UUID
	AdministrationID   uuid.UUID
	SubscaleKey        *string
	RawScore           *float64
	ScaledScore        *float64
	ItemsAnswered      int
	ItemsExpected      int
	BandID             *uuid.UUID
	BandLabel          *string // convenience, hydrated on read via join
	ComputationVersion *string
	ComputedAt         time.Time
	Metadata           json.RawMessage
}

// ScoringEngine computes raw/scaled scores and interpretation bands from a
// catalog (items+rules+bands) and the answers collected for an administration.
// Implementations must be pure/deterministic given the same inputs, so
// results can be reproduced later for audit purposes.
type ScoringEngine interface {
	Compute(items []InstrumentItem, rules []ScoringRule, bands []ScoreBand, answers []AnswerRecord) ([]InstrumentScore, error)
}

// InstrumentAdministrationRepository persists administrations and their scores.
type InstrumentAdministrationRepository interface {
	CreateAdministration(ctx context.Context, projectID, participantID, instrumentID uuid.UUID, instrumentVersion int, clientGeneratedID *string) (*InstrumentAdministration, error)
	GetAdministrationByID(ctx context.Context, administrationID uuid.UUID) (*InstrumentAdministration, error)
	ListAdministrationsByParticipant(ctx context.Context, participantID, instrumentID uuid.UUID) ([]InstrumentAdministration, error)
	UpdateStatus(ctx context.Context, administrationID uuid.UUID, status AdministrationStatus, completedAt *time.Time) (*InstrumentAdministration, error)
	SaveScores(ctx context.Context, administrationID uuid.UUID, scores []InstrumentScore) error
	GetScores(ctx context.Context, administrationID uuid.UUID) ([]InstrumentScore, error)
}

// InstrumentAdministrationUsecase defines the business logic for running and
// scoring instrument applications.
type InstrumentAdministrationUsecase interface {
	StartAdministration(ctx context.Context, actorID, projectID, participantID, instrumentID uuid.UUID, clientGeneratedID *string) (*InstrumentAdministration, error)
	CompleteAdministration(ctx context.Context, actorID, administrationID uuid.UUID) (*InstrumentAdministration, []InstrumentScore, error)
	AbandonAdministration(ctx context.Context, actorID, administrationID uuid.UUID) (*InstrumentAdministration, error)
	ListAdministrations(ctx context.Context, actorID, participantID, instrumentID uuid.UUID) ([]InstrumentAdministration, error)
	GetScores(ctx context.Context, actorID, administrationID uuid.UUID) ([]InstrumentScore, error)
	RecomputeScores(ctx context.Context, actorID, administrationID uuid.UUID) ([]InstrumentScore, error)
}
