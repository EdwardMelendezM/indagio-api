package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.UserInternal, error) {
	var u domain.UserInternal
	err := r.db.QueryRowContext(ctx, queryFindUserByEmail, email).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.Role,
		&u.AvatarURL,
		&u.AvatarVersion,
		&u.Verified,
		&u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return &u, nil
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx, queryFindUserByID, id).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Role,
		&u.AvatarURL,
		&u.AvatarVersion,
		&u.Verified,
		&u.CreatedAt,
		&u.Blocked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("user %s: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &u, nil
}

func (r *postgresUserRepository) Create(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx, queryCreateUser,
		uuid.New(),
		name,
		email,
		passwordHash,
		domain.RoleStudent,
		time.Now().UTC(),
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Role,
		&u.AvatarURL,
		&u.AvatarVersion,
		&u.Verified,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

func (r *postgresUserRepository) MarkVerified(ctx context.Context, email string) error {
	result, err := r.db.ExecContext(ctx, queryMarkUserVerified, email)
	if err != nil {
		return fmt.Errorf("update verified: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("mark verified: user not found: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *postgresUserRepository) UpdateName(ctx context.Context, userID uuid.UUID, name string) error {
	result, err := r.db.ExecContext(ctx, queryUpdateUserName, name, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, email string, passwordHash string) error {
	result, err := r.db.ExecContext(ctx, queryUpdateUserPassword, passwordHash, email)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update password: user not found: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *postgresUserRepository) BlockUser(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, queryBlockUser, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresUserRepository) ListAvailable(
	ctx context.Context,
	excludeUserID uuid.UUID,
	search string,
	limit, offset int,
) ([]domain.User, int, error) {
	// Count first — short-circuit when there are no matches to avoid a second query.
	var total int
	if err := r.db.QueryRowContext(ctx, queryCountAvailableUsers, excludeUserID, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count available users: %w", err)
	}
	if total == 0 {
		return []domain.User{}, 0, nil
	}

	rows, err := r.db.QueryContext(ctx, queryListAvailableUsers, excludeUserID, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list available users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, limit)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.AvatarURL,
			&u.AvatarVersion,
			&u.Verified,
			&u.CreatedAt,
			&u.Blocked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, total, nil
}

// UpdateAvatarURL persists a new avatar JSON document (or clears it
// with nil) and bumps avatar_version atomically. Returns the new
// version so callers can build cache-busted URLs without a second
// round-trip.
func (r *postgresUserRepository) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarJSON *string) (int, error) {
	var (
		value sql.NullString
		ver   int
	)
	if avatarJSON != nil {
		value = sql.NullString{String: *avatarJSON, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, queryUpdateUserAvatarURL, userID, value).Scan(&ver)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("user %s: %w", userID, domain.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("update avatar: %w", err)
	}
	return ver, nil
}

// ClearAvatar wipes the avatar entirely (sets avatar_url = NULL) and
// bumps avatar_version. Returns the new version for cache-busting.
func (r *postgresUserRepository) ClearAvatar(ctx context.Context, userID uuid.UUID) (int, error) {
	var ver int
	err := r.db.QueryRowContext(ctx, queryClearUserAvatar, userID).Scan(&ver)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("user %s: %w", userID, domain.ErrNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("clear avatar: %w", err)
	}
	return ver, nil
}

func (r *postgresUserRepository) ListAll(
	ctx context.Context,
	search string,
	limit, offset int,
) ([]domain.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, queryCountAllUsers, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all users: %w", err)
	}
	if total == 0 {
		return []domain.User{}, 0, nil
	}

	rows, err := r.db.QueryContext(ctx, queryListAllUsers, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list all users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, limit)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.Role,
			&u.AvatarURL,
			&u.AvatarVersion,
			&u.Verified,
			&u.CreatedAt,
			&u.Blocked,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, total, nil
}
