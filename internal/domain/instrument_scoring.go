package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScoringAlgorithm identifies how a set of items combines into a score.
type ScoringAlgorithm string

const (
	ScoringAlgorithmSum         ScoringAlgorithm = "sum"
	ScoringAlgorithmAverage     ScoringAlgorithm = "average"
	ScoringAlgorithmWeightedSum ScoringAlgorithm = "weighted_sum"
	ScoringAlgorithmLookupTable ScoringAlgorithm = "lookup_table"
	ScoringAlgorithmCustom      ScoringAlgorithm = "custom"
)

// MissingStrategy determines how unanswered scored items are handled.
type MissingStrategy string

const (
	MissingStrategyProrate       MissingStrategy = "prorate"
	MissingStrategyZero          MissingStrategy = "zero"
	MissingStrategyNullIfMissing MissingStrategy = "null_if_missing"
)

// InstrumentItem is one scored (or informational) question within a specific
// published version of an instrument's catalog.
type InstrumentItem struct {
	ID                uuid.UUID
	InstrumentID      uuid.UUID
	InstrumentVersion int
	QuestionKey       string
	SubscaleKey       *string
	OrderIndex        int
	ItemType          string
	ReverseScored     bool
	Weight            float64
	ValueMap          json.RawMessage // JSON object: {"label": numericValue, ...}
	IsScored          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// InstrumentSubscale groups items under a named dimension of the instrument.
type InstrumentSubscale struct {
	ID                uuid.UUID
	InstrumentID      uuid.UUID
	InstrumentVersion int
	Key               string
	Name              string
	Description       *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScoringRule defines how the total score, or one subscale's score, is
// computed. SubscaleKey nil means it is the rule for the instrument total.
type ScoringRule struct {
	ID                uuid.UUID
	InstrumentID      uuid.UUID
	InstrumentVersion int
	SubscaleKey       *string
	Algorithm         ScoringAlgorithm
	MinItemsRequired  *int
	MissingStrategy   MissingStrategy
	Config            json.RawMessage
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScoreBand maps a numeric range to an interpretation label (e.g. "leve",
// "moderado", "severo").
type ScoreBand struct {
	ID                uuid.UUID
	InstrumentID      uuid.UUID
	InstrumentVersion int
	SubscaleKey       *string
	MinScore          *float64
	MaxScore          *float64
	Label             string
	Description       *string
	CreatedAt         time.Time
}

// InstrumentCatalog is the full scoring definition for one published version
// of an instrument.
type InstrumentCatalog struct {
	InstrumentID uuid.UUID
	Version      int
	Items        []InstrumentItem
	Subscales    []InstrumentSubscale
	Rules        []ScoringRule
	Bands        []ScoreBand
}

// InstrumentScoringRepository persists the versioned scoring catalog.
// PublishVersion bumps instruments.version transactionally and inserts every
// row tagged with the new version number; it never mutates rows from a
// previous version, so old administrations can always be recomputed against
// the catalog they were originally scored with.
type InstrumentScoringRepository interface {
	PublishVersion(ctx context.Context, instrumentID uuid.UUID, items []InstrumentItem, subscales []InstrumentSubscale, rules []ScoringRule, bands []ScoreBand) (*InstrumentCatalog, error)
	GetCatalogByVersion(ctx context.Context, instrumentID uuid.UUID, version int) (*InstrumentCatalog, error)
	GetLatestCatalog(ctx context.Context, instrumentID uuid.UUID) (*InstrumentCatalog, error)
	ListVersions(ctx context.Context, instrumentID uuid.UUID) ([]int, error)
}

// InstrumentScoringUsecase defines the business logic for publishing and
// reading an instrument's scoring catalog.
type InstrumentScoringUsecase interface {
	PublishVersion(ctx context.Context, actorID, instrumentID uuid.UUID, items []InstrumentItem, subscales []InstrumentSubscale, rules []ScoringRule, bands []ScoreBand) (*InstrumentCatalog, error)
	GetCatalog(ctx context.Context, actorID, instrumentID uuid.UUID, version *int) (*InstrumentCatalog, error)
	ListVersions(ctx context.Context, actorID, instrumentID uuid.UUID) ([]int, error)
}
