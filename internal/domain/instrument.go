package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InstrumentKind describes the runtime behavior of a reusable instrument.
type InstrumentKind string

const (
	InstrumentKindText         InstrumentKind = "text"
	InstrumentKindNumeric      InstrumentKind = "numeric"
	InstrumentKindScale        InstrumentKind = "scale"
	InstrumentKindSingleChoice InstrumentKind = "single_choice"
	InstrumentKindMultiChoice  InstrumentKind = "multi_choice"
	InstrumentKindAudio        InstrumentKind = "audio"
	InstrumentKindVideo        InstrumentKind = "video"
)

// InstrumentStatus identifies the lifecycle state of an instrument.
type InstrumentStatus string

const (
	InstrumentStatusDraft    InstrumentStatus = "draft"
	InstrumentStatusActive   InstrumentStatus = "active"
	InstrumentStatusArchived InstrumentStatus = "archived"
)

// Instrument is a reusable item template for a project.
type Instrument struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Kind      InstrumentKind
	Config    json.RawMessage
	Version   int
	Status    InstrumentStatus
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Questionnaire is a versioned project questionnaire schema.
type Questionnaire struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	Version   int
	Status    string
	Schema    json.RawMessage
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InstrumentRepository stores instrument templates and questionnaire schemas.
type InstrumentRepository interface {
	CreateInstrument(ctx context.Context, projectID, createdBy uuid.UUID, name string, kind InstrumentKind, config json.RawMessage) (*Instrument, error)
	GetInstrumentByID(ctx context.Context, instrumentID uuid.UUID) (*Instrument, error)
	ListInstrumentsByProject(ctx context.Context, projectID uuid.UUID) ([]Instrument, error)
	UpdateInstrument(ctx context.Context, instrumentID uuid.UUID, name *string, config *json.RawMessage, status *InstrumentStatus) (*Instrument, error)
}

// InstrumentUsecase defines the business logic for instrument/questionnaire management.
type InstrumentUsecase interface {
	CreateInstrument(ctx context.Context, actorID, projectID uuid.UUID, name string, kind string, config json.RawMessage) (*Instrument, error)
	ListInstruments(ctx context.Context, actorID, projectID uuid.UUID) ([]Instrument, error)
	UpdateInstrument(ctx context.Context, actorID, instrumentID uuid.UUID, name *string, config *json.RawMessage, status *InstrumentStatus) (*Instrument, error)
}
