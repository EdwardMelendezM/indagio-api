package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"foro-unsaac-backend/internal/domain"
)

type postgresOTPRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) domain.OTPRepository {
	return &postgresOTPRepository{db: db}
}

func (r *postgresOTPRepository) Create(ctx context.Context, email, code string) error {
	expiresAt := time.Now().UTC().Add(15 * time.Minute)

	_, err := r.db.ExecContext(ctx, queryCreateOTP, email, code, expiresAt)
	if err != nil {
		return fmt.Errorf("insert otp: %w", err)
	}

	return nil
}

func (r *postgresOTPRepository) ValidateAndConsume(ctx context.Context, email, code string) (bool, error) {
	var stored string
	var expiresAt time.Time

	err := r.db.QueryRowContext(ctx, queryFindActiveOTP, email).Scan(&stored, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query otp: %w", err)
	}

	if time.Now().UTC().After(expiresAt) {
		return false, nil
	}

	if stored != code {
		return false, nil
	}

	if _, err = r.db.ExecContext(ctx, queryConsumeOTP, email); err != nil {
		return false, fmt.Errorf("consume otp: %w", err)
	}

	return true, nil
}
