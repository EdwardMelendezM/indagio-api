package participants

import (
	"context"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubParticipantRepo struct {
	participants map[uuid.UUID]*domain.Participant
	byCode       map[string]uuid.UUID
}

func (s *stubParticipantRepo) Create(ctx context.Context, projectID uuid.UUID, code, displayName, externalIdentifier string) (*domain.Participant, error) {
	participant := &domain.Participant{
		ID:                 uuid.New(),
		ProjectID:          projectID,
		Code:               code,
		DisplayName:        displayName,
		ExternalIdentifier: externalIdentifier,
		Status:             domain.ParticipantStatusPending,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	s.participants[participant.ID] = participant
	s.byCode[code] = participant.ID
	return participant, nil
}

func (s *stubParticipantRepo) GetByID(ctx context.Context, participantID uuid.UUID) (*domain.Participant, error) {
	p, ok := s.participants[participantID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *stubParticipantRepo) GetByCode(ctx context.Context, projectID uuid.UUID, code string) (*domain.Participant, error) {
	id, ok := s.byCode[code]
	if !ok {
		return nil, domain.ErrNotFound
	}
	p, ok := s.participants[id]
	if !ok || p.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *stubParticipantRepo) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Participant, error) {
	out := make([]domain.Participant, 0)
	for _, p := range s.participants {
		if p.ProjectID == projectID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (s *stubParticipantRepo) UpdateStatus(ctx context.Context, participantID uuid.UUID, status domain.ParticipantStatus) (*domain.Participant, error) {
	p, ok := s.participants[participantID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

func TestParticipantUsecase_CreateParticipant_GeneratesCode(t *testing.T) {
	projectID := uuid.New()
	actorID := uuid.New()
	repo := &stubParticipantRepo{participants: map[uuid.UUID]*domain.Participant{}, byCode: map[string]uuid.UUID{}}
	uc := NewParticipantUsecase(repo)

	participant, err := uc.CreateParticipant(context.Background(), actorID, projectID, "Alice Jones", "ext-001")
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if participant.Code == "" {
		t.Fatal("expected code to be generated")
	}
	if participant.Status != domain.ParticipantStatusPending {
		t.Fatalf("expected pending status, got %s", participant.Status)
	}
}

func TestParticipantUsecase_GetParticipantByCode_RejectsEmptyCode(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	repo := &stubParticipantRepo{participants: map[uuid.UUID]*domain.Participant{}, byCode: map[string]uuid.UUID{}}
	uc := NewParticipantUsecase(repo)

	_, err := uc.GetParticipantByCode(context.Background(), actorID, projectID, " ")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
