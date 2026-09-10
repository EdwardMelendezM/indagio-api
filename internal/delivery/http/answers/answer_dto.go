package answers

import (
	"encoding/json"
	"time"

	"indagio-api/internal/domain"
)

type CreateAnswerRequest struct {
	ParticipantID     string          `json:"participant_id" binding:"required"`
	InstrumentID      *string         `json:"instrument_id"`
	QuestionnaireID   *string         `json:"questionnaire_id"`
	QuestionKey       string          `json:"question_key" binding:"required"`
	AnswerType        string          `json:"answer_type" binding:"required"`
	Value             json.RawMessage `json:"value" binding:"required"`
	ClientGeneratedID *string         `json:"client_generated_id"`
}

type CreateMediaRequest struct {
	AnswerID        *string `json:"answer_id"`
	FileKey         string  `json:"file_key" binding:"required"`
	MimeType        string  `json:"mime_type" binding:"required"`
	SizeBytes       int64   `json:"size_bytes" binding:"required"`
	DurationSeconds *int    `json:"duration_seconds"`
	Checksum        *string `json:"checksum"`
}

type AnswerResponse struct {
	ID                string          `json:"id"`
	ProjectID         string          `json:"project_id"`
	ParticipantID     string          `json:"participant_id"`
	InstrumentID      *string         `json:"instrument_id,omitempty"`
	QuestionnaireID   *string         `json:"questionnaire_id,omitempty"`
	QuestionKey       string          `json:"question_key"`
	AnswerType        string          `json:"answer_type"`
	Value             json.RawMessage `json:"value"`
	Status            string          `json:"status"`
	SyncStatus        string          `json:"sync_status"`
	ClientGeneratedID *string         `json:"client_generated_id,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type MediaResponse struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	ParticipantID   string    `json:"participant_id"`
	AnswerID        *string   `json:"answer_id,omitempty"`
	FileKey         string    `json:"file_key"`
	MimeType        string    `json:"mime_type"`
	SizeBytes       int64     `json:"size_bytes"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	Checksum        *string   `json:"checksum,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func ToAnswerResponse(a *domain.AnswerRecord) AnswerResponse {
	resp := AnswerResponse{
		ID:            a.ID.String(),
		ProjectID:     a.ProjectID.String(),
		ParticipantID: a.ParticipantID.String(),
		QuestionKey:   a.QuestionKey,
		AnswerType:    string(a.AnswerType),
		Value:         a.Value,
		Status:        string(a.Status),
		SyncStatus:    string(a.SyncStatus),
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
	if a.InstrumentID != nil {
		v := a.InstrumentID.String()
		resp.InstrumentID = &v
	}
	if a.QuestionnaireID != nil {
		v := a.QuestionnaireID.String()
		resp.QuestionnaireID = &v
	}
	if a.ClientGeneratedID != nil {
		resp.ClientGeneratedID = a.ClientGeneratedID
	}
	return resp
}

func ToMediaResponse(m *domain.MediaFile) MediaResponse {
	resp := MediaResponse{
		ID:            m.ID.String(),
		ProjectID:     m.ProjectID.String(),
		ParticipantID: m.ParticipantID.String(),
		FileKey:       m.FileKey,
		MimeType:      m.MimeType,
		SizeBytes:     m.SizeBytes,
		Status:        m.Status,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.AnswerID != nil {
		v := m.AnswerID.String()
		resp.AnswerID = &v
	}
	if m.DurationSeconds != nil {
		resp.DurationSeconds = m.DurationSeconds
	}
	if m.Checksum != nil {
		resp.Checksum = m.Checksum
	}
	return resp
}
