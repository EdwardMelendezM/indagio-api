package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AnswerType identifies the data model used by a single answer record.
type AnswerType string

const (
	AnswerTypeSingleChoice   AnswerType = "single_choice"
	AnswerTypeMultipleChoice AnswerType = "multiple_choice"
	AnswerTypeText           AnswerType = "text"
	AnswerTypeScale          AnswerType = "scale"
)

// AnswerStatus indicates the lifecycle of an answer record.
type AnswerStatus string

const (
	AnswerStatusDraft  AnswerStatus = "draft"
	AnswerStatusSaved  AnswerStatus = "saved"
	AnswerStatusSynced AnswerStatus = "synced"
)

// SyncStatus marks whether a record is pending upload/sync or already reconciled.
type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusFailed  SyncStatus = "failed"
)

// AnswerRecord is a project-level response submitted by a participant.
type AnswerRecord struct {
	ID                uuid.UUID
	ProjectID         uuid.UUID
	ParticipantID     uuid.UUID
	InstrumentID      *uuid.UUID
	Instrument        *Instrument
	QuestionKey       string
	AnswerType        AnswerType
	Value             json.RawMessage
	Status            AnswerStatus
	SyncStatus        SyncStatus
	ClientGeneratedID *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// MediaFile is a persisted artifact uploaded for an answer.
type MediaFile struct {
	ID              uuid.UUID
	ProjectID       uuid.UUID
	ParticipantID   uuid.UUID
	AnswerID        *uuid.UUID
	FileKey         string
	MimeType        string
	SizeBytes       int64
	DurationSeconds *int
	Checksum        *string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AnswerRepository is the persistence contract for answer records and media.
type AnswerRepository interface {
	CreateAnswer(ctx context.Context, projectID, participantID uuid.UUID, instrumentID *uuid.UUID, questionKey string, answerType AnswerType, value json.RawMessage, clientGeneratedID *string) (*AnswerRecord, error)
	GetAnswerByID(ctx context.Context, answerID uuid.UUID) (*AnswerRecord, error)
	ListAnswersByParticipant(ctx context.Context, participantID uuid.UUID, instrumentID *uuid.UUID) ([]AnswerRecord, error)
	UpsertAnswer(ctx context.Context, projectID, participantID uuid.UUID, answerType AnswerType, questionKey string, value json.RawMessage, clientGeneratedID string) (*AnswerRecord, error)
	CreateMedia(ctx context.Context, projectID, participantID uuid.UUID, answerID *uuid.UUID, fileKey, mimeType string, sizeBytes int64, durationSeconds *int, checksum *string) (*MediaFile, error)
	ListMediaByAnswer(ctx context.Context, answerID uuid.UUID) ([]MediaFile, error)
}

// AnswerUsecase defines the business logic for answer storage and media workflows.
type AnswerUsecase interface {
	CreateAnswer(ctx context.Context, actorID, projectID, participantID uuid.UUID, instrumentID *uuid.UUID, questionKey string, answerType string, value json.RawMessage, clientGeneratedID *string) (*AnswerRecord, error)
	ListAnswers(ctx context.Context, actorID, projectID, participantID uuid.UUID, instrumentID *uuid.UUID) ([]AnswerRecord, error)
	CreateMedia(ctx context.Context, actorID, projectID, participantID uuid.UUID, answerID *uuid.UUID, fileKey, mimeType string, sizeBytes int64, durationSeconds *int, checksum *string) (*MediaFile, error)
}
