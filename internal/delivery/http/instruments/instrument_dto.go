package instruments

import (
	"encoding/json"
	"time"

	"indagio-api/internal/domain"
)

type CreateInstrumentRequest struct {
	Name   string          `json:"name" binding:"required,min=2,max=160"`
	Kind   string          `json:"kind" binding:"required"`
	Config json.RawMessage `json:"config" swaggertype:"object"`
}

type UpdateInstrumentRequest struct {
	Name   *string          `json:"name"`
	Config *json.RawMessage `json:"config" swaggertype:"object"`
	Status *string          `json:"status"`
}

type CreateQuestionnaireRequest struct {
	Name   string          `json:"name" binding:"required,min=2,max=160"`
	Schema json.RawMessage `json:"schema" binding:"required" swaggertype:"object"`
}

type InstrumentResponse struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Config    json.RawMessage `json:"config" swaggertype:"object"`
	Version   int             `json:"version"`
	Status    string          `json:"status"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type QuestionnaireResponse struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	Version   int             `json:"version"`
	Status    string          `json:"status"`
	Schema    json.RawMessage `json:"schema" swaggertype:"object"`
	CreatedBy string          `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func ToInstrumentResponse(inst *domain.Instrument) InstrumentResponse {
	return InstrumentResponse{
		ID:        inst.ID.String(),
		ProjectID: inst.ProjectID.String(),
		Name:      inst.Name,
		Kind:      string(inst.Kind),
		Config:    inst.Config,
		Version:   inst.Version,
		Status:    string(inst.Status),
		CreatedBy: inst.CreatedBy.String(),
		CreatedAt: inst.CreatedAt,
		UpdatedAt: inst.UpdatedAt,
	}
}

func ToQuestionnaireResponse(q *domain.Questionnaire) QuestionnaireResponse {
	return QuestionnaireResponse{
		ID:        q.ID.String(),
		ProjectID: q.ProjectID.String(),
		Name:      q.Name,
		Version:   q.Version,
		Status:    q.Status,
		Schema:    q.Schema,
		CreatedBy: q.CreatedBy.String(),
		CreatedAt: q.CreatedAt,
		UpdatedAt: q.UpdatedAt,
	}
}
