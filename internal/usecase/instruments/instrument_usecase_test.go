package instruments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubInstrumentRepo struct {
	instruments    map[uuid.UUID]*domain.Instrument
	questionnaires map[uuid.UUID]*domain.Questionnaire
}

func (s *stubInstrumentRepo) CreateInstrument(ctx context.Context, projectID, createdBy uuid.UUID, name string, kind domain.InstrumentKind, config json.RawMessage) (*domain.Instrument, error) {
	instrument := &domain.Instrument{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		Kind:      kind,
		Config:    config,
		Version:   1,
		Status:    domain.InstrumentStatusDraft,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.instruments[instrument.ID] = instrument
	return instrument, nil
}

func (s *stubInstrumentRepo) GetInstrumentByID(ctx context.Context, instrumentID uuid.UUID) (*domain.Instrument, error) {
	instrument, ok := s.instruments[instrumentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return instrument, nil
}

func (s *stubInstrumentRepo) ListInstrumentsByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Instrument, error) {
	out := make([]domain.Instrument, 0)
	for _, instrument := range s.instruments {
		if instrument.ProjectID == projectID {
			out = append(out, *instrument)
		}
	}
	return out, nil
}

func (s *stubInstrumentRepo) UpdateInstrument(ctx context.Context, instrumentID uuid.UUID, name *string, config *json.RawMessage, status *domain.InstrumentStatus) (*domain.Instrument, error) {
	instrument, ok := s.instruments[instrumentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if name != nil {
		instrument.Name = *name
	}
	if config != nil {
		instrument.Config = *config
	}
	if status != nil {
		instrument.Status = *status
	}
	instrument.UpdatedAt = time.Now().UTC()
	return instrument, nil
}

func (s *stubInstrumentRepo) CreateQuestionnaire(ctx context.Context, projectID, createdBy uuid.UUID, name string, schema json.RawMessage) (*domain.Questionnaire, error) {
	questionnaire := &domain.Questionnaire{
		ID:        uuid.New(),
		ProjectID: projectID,
		Name:      name,
		Version:   1,
		Status:    "draft",
		Schema:    schema,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.questionnaires[questionnaire.ID] = questionnaire
	return questionnaire, nil
}

func (s *stubInstrumentRepo) ListQuestionnairesByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Questionnaire, error) {
	out := make([]domain.Questionnaire, 0)
	for _, questionnaire := range s.questionnaires {
		if questionnaire.ProjectID == projectID {
			out = append(out, *questionnaire)
		}
	}
	return out, nil
}

func (s *stubInstrumentRepo) GetQuestionnaireByID(ctx context.Context, questionnaireID uuid.UUID) (*domain.Questionnaire, error) {
	questionnaire, ok := s.questionnaires[questionnaireID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return questionnaire, nil
}

func (s *stubInstrumentRepo) PublishQuestionnaire(ctx context.Context, questionnaireID uuid.UUID) (*domain.Questionnaire, error) {
	questionnaire, ok := s.questionnaires[questionnaireID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	questionnaire.Status = "published"
	questionnaire.UpdatedAt = time.Now().UTC()
	return questionnaire, nil
}

func TestInstrumentUsecase_CreateInstrument_ValidatesJSONAndKind(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	repo := &stubInstrumentRepo{instruments: map[uuid.UUID]*domain.Instrument{}, questionnaires: map[uuid.UUID]*domain.Questionnaire{}}
	uc := NewInstrumentUsecase(repo)

	_, err := uc.CreateInstrument(context.Background(), actorID, projectID, "Mood scale", "not-valid", json.RawMessage(`{"min":0}`))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid kind, got %v", err)
	}
}

func TestInstrumentUsecase_CreateQuestionnaire_RequiresValidSchema(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	repo := &stubInstrumentRepo{instruments: map[uuid.UUID]*domain.Instrument{}, questionnaires: map[uuid.UUID]*domain.Questionnaire{}}
	uc := NewInstrumentUsecase(repo)

	_, err := uc.CreateQuestionnaire(context.Background(), actorID, projectID, "Daily form", json.RawMessage(`not-json`))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation for invalid schema, got %v", err)
	}
}
