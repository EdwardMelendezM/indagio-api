package answers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubAnswerRepo struct {
	answers map[uuid.UUID]*domain.AnswerRecord
	media   map[uuid.UUID]*domain.MediaFile
}

func (s *stubAnswerRepo) CreateAnswer(ctx context.Context, projectID, participantID uuid.UUID, instrumentID, questionnaireID *uuid.UUID, questionKey string, answerType domain.AnswerType, value json.RawMessage, clientGeneratedID *string) (*domain.AnswerRecord, error) {
	answer := &domain.AnswerRecord{
		ID:                uuid.New(),
		ProjectID:         projectID,
		ParticipantID:     participantID,
		InstrumentID:      instrumentID,
		QuestionnaireID:   questionnaireID,
		QuestionKey:       questionKey,
		AnswerType:        answerType,
		Value:             value,
		Status:            domain.AnswerStatusDraft,
		SyncStatus:        domain.SyncStatusPending,
		ClientGeneratedID: clientGeneratedID,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	s.answers[answer.ID] = answer
	return answer, nil
}

func (s *stubAnswerRepo) GetAnswerByID(ctx context.Context, answerID uuid.UUID) (*domain.AnswerRecord, error) {
	answer, ok := s.answers[answerID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return answer, nil
}

func (s *stubAnswerRepo) ListAnswersByParticipant(ctx context.Context, participantID uuid.UUID) ([]domain.AnswerRecord, error) {
	out := make([]domain.AnswerRecord, 0)
	for _, answer := range s.answers {
		if answer.ParticipantID == participantID {
			out = append(out, *answer)
		}
	}
	return out, nil
}

func (s *stubAnswerRepo) UpsertAnswer(ctx context.Context, projectID, participantID uuid.UUID, answerType domain.AnswerType, questionKey string, value json.RawMessage, clientGeneratedID string) (*domain.AnswerRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAnswerRepo) CreateMedia(ctx context.Context, projectID, participantID uuid.UUID, answerID *uuid.UUID, fileKey, mimeType string, sizeBytes int64, durationSeconds *int, checksum *string) (*domain.MediaFile, error) {
	media := &domain.MediaFile{
		ID:              uuid.New(),
		ProjectID:       projectID,
		ParticipantID:   participantID,
		AnswerID:        answerID,
		FileKey:         fileKey,
		MimeType:        mimeType,
		SizeBytes:       sizeBytes,
		DurationSeconds: durationSeconds,
		Checksum:        checksum,
		Status:          "pending",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	s.media[media.ID] = media
	return media, nil
}

func (s *stubAnswerRepo) ListMediaByAnswer(ctx context.Context, answerID uuid.UUID) ([]domain.MediaFile, error) {
	return nil, nil
}

func TestAnswerUsecase_CreateAnswer_RequiresJSONValue(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	participantID := uuid.New()
	repo := &stubAnswerRepo{answers: map[uuid.UUID]*domain.AnswerRecord{}, media: map[uuid.UUID]*domain.MediaFile{}}
	uc := NewAnswerUsecase(repo)
	_, err := uc.CreateAnswer(context.Background(), actorID, projectID, participantID, nil, nil, "q1", "text", json.RawMessage(`not-json`), nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestAnswerUsecase_CreateMedia_RequiresFileMetadata(t *testing.T) {
	actorID := uuid.New()
	projectID := uuid.New()
	participantID := uuid.New()
	repo := &stubAnswerRepo{answers: map[uuid.UUID]*domain.AnswerRecord{}, media: map[uuid.UUID]*domain.MediaFile{}}
	uc := NewAnswerUsecase(repo)
	_, err := uc.CreateMedia(context.Background(), actorID, projectID, participantID, nil, "", "image/png", 10, nil, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
