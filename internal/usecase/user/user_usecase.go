package user

import (
	"context"
	"fmt"
	"strings"

	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

// UserUsecase orchestrates user-related business logic. It currently
// owns the "list users available to start a conversation with" use case
// — kept separate from the auth usecase to preserve single responsibility.
type UserUsecase struct {
	userRepo domain.UserRepository
}

// NewUserUsecase wires the usecase with its only dependency.
func NewUserUsecase(userRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{userRepo: userRepo}
}

// ListAvailable returns verified, non-deleted users (excluding the caller)
// that the caller can start a conversation with. Supports an optional
// case-insensitive search across name/email. Pagination is clamped to the
// same defaults the conversation endpoints use (page>=1, 1<=limit<=100).
func (u *UserUsecase) ListAvailable(
	ctx context.Context,
	currentUserID uuid.UUID,
	search string,
	page, limit int,
) ([]domain.User, int, error) {
	// Defensive clamping — the Gin binding already enforces these but the
	// usecase must remain correct when called from other contexts (tests,
	// future gRPC handlers, etc.).
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	search = strings.TrimSpace(search)
	if len(search) > 80 {
		search = search[:80]
	}

	offset := (page - 1) * limit
	users, total, err := u.userRepo.ListAvailable(ctx, currentUserID, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list available users: %w", err)
	}
	return users, total, nil
}

// ListAll returns all verified, non-deleted users (including blocked) for admin management.
// Supports case-insensitive search across name/email. Pagination is clamped to
// page>=1, 1<=limit<=100.
func (u *UserUsecase) ListAll(
	ctx context.Context,
	search string,
	page, limit int,
) ([]domain.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	search = strings.TrimSpace(search)
	if len(search) > 80 {
		search = search[:80]
	}

	offset := (page - 1) * limit
	users, total, err := u.userRepo.ListAll(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list all users: %w", err)
	}
	return users, total, nil
}
