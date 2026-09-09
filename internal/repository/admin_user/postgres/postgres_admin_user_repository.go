package admin_users_postgres

import (
	"context"
	"database/sql"
	"errors"

	"foro-unsaac-backend/internal/domain"

	"github.com/google/uuid"
)

type postgresAdminUserRepository struct {
	db *sql.DB
}

// NewAdminUserRepository constructs a repository that satisfies domain.AdminUserRepository.
func NewAdminUserRepository(db *sql.DB) domain.AdminUserRepository {
	return &postgresAdminUserRepository{db: db}
}

// FindByEmail returns the active admin (deleted_at IS NULL) matching the given email.
// The partial unique index on the table guarantees at most one active row per email.
func (r *postgresAdminUserRepository) FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, queryFindByEmail, email)

	var (
		rawID     string
		adminUser domain.AdminUser
	)

	err := row.Scan(
		&rawID,
		&adminUser.Email,
		&adminUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, err
	}
	adminUser.ID = parsedID

	return &adminUser, nil
}

// FindById returns the active admin (deleted_at IS NULL) matching the given id.
func (r *postgresAdminUserRepository) FindById(ctx context.Context, id uuid.UUID) (*domain.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, queryFindById, id)

	var (
		rawID     string
		adminUser domain.AdminUser
	)

	err := row.Scan(
		&rawID,
		&adminUser.Email,
		&adminUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, err
	}
	adminUser.ID = parsedID

	return &adminUser, nil
}

// MarkVerified sets deleted_at to current timestamp for the given email, effectively marking the OTP as used.
func (r *postgresAdminUserRepository) MarkVerified(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, queryMarkVerified, email)
	return err
}

// Create inserts a new admin user with the given email. This is not exposed via any public API, but can be used internally to seed the initial admin.
func (r *postgresAdminUserRepository) Create(ctx context.Context, email string) (*domain.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, queryCreate, email)

	var (
		rawID     string
		adminUser domain.AdminUser
	)

	err := row.Scan(
		&rawID,
		&adminUser.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, err
	}
	adminUser.ID = parsedID
	adminUser.Email = email

	return &adminUser, nil
}

// FindFirst returns the first active admin user. This can be used to seed the initial admin if no users exist.
func (r *postgresAdminUserRepository) FindFirst(ctx context.Context) (*domain.AdminUser, error) {
	row := r.db.QueryRowContext(ctx, queryFindFirst)

	var (
		rawID     string
		adminUser domain.AdminUser
	)

	err := row.Scan(
		&rawID,
		&adminUser.Email,
		&adminUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, err
	}
	adminUser.ID = parsedID

	return &adminUser, nil
}
