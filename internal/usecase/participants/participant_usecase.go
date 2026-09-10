package participants

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type participantUsecase struct {
	repo domain.ParticipantRepository
}

func NewParticipantUsecase(repo domain.ParticipantRepository) domain.ParticipantUsecase {
	return &participantUsecase{repo: repo}
}

func (uc *participantUsecase) CreateParticipant(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, displayName, externalIdentifier string) (*domain.Participant, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	displayName = strings.TrimSpace(displayName)
	externalIdentifier = strings.TrimSpace(externalIdentifier)
	if len(displayName) < 2 || len(displayName) > 160 {
		return nil, fmt.Errorf("display name: %w", domain.ErrValidation)
	}
	if externalIdentifier != "" && len(externalIdentifier) > 160 {
		return nil, fmt.Errorf("external identifier: %w", domain.ErrValidation)
	}

	code := generateParticipantCode()
	participant, err := uc.repo.Create(ctx, projectID, code, displayName, externalIdentifier)
	if err != nil {
		return nil, fmt.Errorf("create participant: %w", err)
	}
	return participant, nil
}

func (uc *participantUsecase) ListParticipants(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) ([]domain.Participant, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	participants, err := uc.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}
	return participants, nil
}

func (uc *participantUsecase) GetParticipantByCode(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, code string) (*domain.Participant, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, fmt.Errorf("code: %w", domain.ErrValidation)
	}
	participant, err := uc.repo.GetByCode(ctx, projectID, code)
	if err != nil {
		return nil, fmt.Errorf("find participant by code: %w", err)
	}
	return participant, nil
}

func (uc *participantUsecase) UpdateParticipantStatus(ctx context.Context, actorID uuid.UUID, participantID uuid.UUID, status domain.ParticipantStatus) (*domain.Participant, error) {
	if actorID == uuid.Nil || participantID == uuid.Nil {
		return nil, fmt.Errorf("actor/participant id: %w", domain.ErrValidation)
	}
	if status != domain.ParticipantStatusPending && status != domain.ParticipantStatusActive && status != domain.ParticipantStatusCompleted && status != domain.ParticipantStatusExpired {
		return nil, fmt.Errorf("status: %w", domain.ErrValidation)
	}

	participant, err := uc.repo.UpdateStatus(ctx, participantID, status)
	if err != nil {
		return nil, fmt.Errorf("update participant status: %w", err)
	}
	return participant, nil
}

func generateParticipantCode() string {
	alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const length = 8
	var sb strings.Builder
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return fmt.Sprintf("CODE-%d", time.Now().UnixNano()%1000000)
		}
		sb.WriteByte(alphabet[idx.Int64()])
	}
	return sb.String()
}
