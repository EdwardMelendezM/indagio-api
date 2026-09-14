package answers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type answerUsecase struct {
	repo domain.AnswerRepository
}

func NewAnswerUsecase(repo domain.AnswerRepository) domain.AnswerUsecase {
	return &answerUsecase{repo: repo}
}

func (uc *answerUsecase) CreateAnswer(ctx context.Context, actorID, projectID, participantID uuid.UUID, instrumentID, questionnaireID *uuid.UUID, questionKey string, answerType string, value json.RawMessage, clientGeneratedID *string) (*domain.AnswerRecord, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/participant ids: %w", domain.ErrValidation)
	}
	questionKey = strings.TrimSpace(questionKey)
	if questionKey == "" {
		return nil, fmt.Errorf("question key: %w", domain.ErrValidation)
	}
	if !json.Valid(value) {
		return nil, fmt.Errorf("value: %w", domain.ErrValidation)
	}
	parsedType := domain.AnswerType(answerType)
	if parsedType != domain.AnswerTypeSingleChoice && parsedType != domain.AnswerTypeMultipleChoice && parsedType != domain.AnswerTypeText && parsedType != domain.AnswerTypeScale {
		return nil, fmt.Errorf("answer type: %w", domain.ErrValidation)
	}
	return uc.repo.CreateAnswer(ctx, projectID, participantID, instrumentID, questionnaireID, questionKey, parsedType, value, clientGeneratedID)
}

func (uc *answerUsecase) ListAnswers(ctx context.Context, actorID, projectID, participantID uuid.UUID) ([]domain.AnswerRecord, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/participant ids: %w", domain.ErrValidation)
	}
	return uc.repo.ListAnswersByParticipant(ctx, participantID)
}

func (uc *answerUsecase) CreateMedia(ctx context.Context, actorID, projectID, participantID uuid.UUID, answerID *uuid.UUID, fileKey, mimeType string, sizeBytes int64, durationSeconds *int, checksum *string) (*domain.MediaFile, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/participant ids: %w", domain.ErrValidation)
	}
	fileKey = strings.TrimSpace(fileKey)
	mimeType = strings.TrimSpace(mimeType)
	if fileKey == "" || mimeType == "" {
		return nil, fmt.Errorf("file metadata: %w", domain.ErrValidation)
	}
	if sizeBytes < 0 {
		return nil, fmt.Errorf("size bytes: %w", domain.ErrValidation)
	}
	return uc.repo.CreateMedia(ctx, projectID, participantID, answerID, fileKey, mimeType, sizeBytes, durationSeconds, checksum)
}
