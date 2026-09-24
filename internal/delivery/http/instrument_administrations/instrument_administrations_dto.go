package instrument_administrations

import (
	"encoding/json"
	"time"

	"indagio-api/internal/domain"
)

type StartAdministrationRequest struct {
	ClientGeneratedID *string `json:"client_generated_id"`
}

type AdministrationResponse struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	ParticipantID     string     `json:"participant_id"`
	InstrumentID      string     `json:"instrument_id"`
	InstrumentVersion int        `json:"instrument_version"`
	Status            string     `json:"status"`
	SyncStatus        string     `json:"sync_status"`
	ClientGeneratedID *string    `json:"client_generated_id"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
}

type ScoreResponse struct {
	SubscaleKey        *string         `json:"subscale_key"`
	RawScore           *float64        `json:"raw_score"`
	ScaledScore        *float64        `json:"scaled_score"`
	ItemsAnswered      int             `json:"items_answered"`
	ItemsExpected      int             `json:"items_expected"`
	BandLabel          *string         `json:"band_label"`
	ComputationVersion *string         `json:"computation_version"`
	ComputedAt         time.Time       `json:"computed_at"`
	Metadata           json.RawMessage `json:"metadata" swaggertype:"object"`
}

type CompleteAdministrationResponse struct {
	Administration AdministrationResponse `json:"administration"`
	Scores         []ScoreResponse        `json:"scores"`
}

func ToAdministrationResponse(a *domain.InstrumentAdministration) AdministrationResponse {
	return AdministrationResponse{
		ID: a.ID.String(), ProjectID: a.ProjectID.String(), ParticipantID: a.ParticipantID.String(),
		InstrumentID: a.InstrumentID.String(), InstrumentVersion: a.InstrumentVersion,
		Status: string(a.Status), SyncStatus: string(a.SyncStatus), ClientGeneratedID: a.ClientGeneratedID,
		StartedAt: a.StartedAt, CompletedAt: a.CompletedAt,
	}
}

func ToScoreResponses(scores []domain.InstrumentScore) []ScoreResponse {
	resp := make([]ScoreResponse, 0, len(scores))
	for _, s := range scores {
		resp = append(resp, ScoreResponse{
			SubscaleKey: s.SubscaleKey, RawScore: s.RawScore, ScaledScore: s.ScaledScore,
			ItemsAnswered: s.ItemsAnswered, ItemsExpected: s.ItemsExpected, BandLabel: s.BandLabel,
			ComputationVersion: s.ComputationVersion, ComputedAt: s.ComputedAt, Metadata: s.Metadata,
		})
	}
	return resp
}
