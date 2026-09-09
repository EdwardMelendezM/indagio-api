package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"indagio-api/internal/domain"
)

func setupUserTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("unable to create mock db: %v", err)
	}
	return db, mock
}

// TestUserRepository_Create tests user creation
func TestUserRepository_Create(t *testing.T) {
	t.Run("When create user successfully", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()
		name := "John Doe"
		email := "john@unsaac.edu.pe"
		passwordHash := "$2a$12$R9h7cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUe"

		rows := sqlmock.NewRows([]string{"id", "name", "email", "role", "avatar_url", "avatar_version", "verified", "created_at"}).
			AddRow(userID, name, email, "estudiante", nil, 0, false, time.Now().UTC())

		mock.ExpectQuery(queryCreateUser).
			WithArgs(sqlmock.AnyArg(), name, email, passwordHash, "estudiante", sqlmock.AnyArg()).
			WillReturnRows(rows)

		repo := NewUserRepository(db)
		user, err := repo.Create(context.Background(), name, email, passwordHash)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, name, user.Name)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, false, user.Verified)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When create user fails with db error", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		name := "John Doe"
		email := "john@unsaac.edu.pe"
		passwordHash := "$2a$12$hash"

		mock.ExpectQuery(queryCreateUser).
			WithArgs(sqlmock.AnyArg(), name, email, passwordHash, "estudiante", sqlmock.AnyArg()).
			WillReturnError(errors.New("unique constraint violation"))

		repo := NewUserRepository(db)
		user, err := repo.Create(context.Background(), name, email, passwordHash)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "insert user")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUserRepository_FindByEmail tests finding user by email
func TestUserRepository_FindByEmail(t *testing.T) {
	t.Run("When find user by email successfully", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()
		name := "Jane Doe"
		email := "jane@unsaac.edu.pe"
		passwordHash := "$2a$12$hash"
		role := domain.RoleStudent
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "role", "avatar_url", "avatar_version", "verified", "created_at"}).
			AddRow(userID, name, email, passwordHash, role, nil, 0, true, now)

		mock.ExpectQuery(queryFindUserByEmail).
			WithArgs(email).
			WillReturnRows(rows)

		repo := NewUserRepository(db)
		user, err := repo.FindByEmail(context.Background(), email)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, name, user.Name)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, passwordHash, user.Password)
		assert.Equal(t, true, user.Verified)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When user not found", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "nonexistent@unsaac.edu.pe"

		mock.ExpectQuery(queryFindUserByEmail).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		repo := NewUserRepository(db)
		user, err := repo.FindByEmail(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, domain.IsNotFound(err))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When database error occurs", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"

		mock.ExpectQuery(queryFindUserByEmail).
			WithArgs(email).
			WillReturnError(errors.New("connection timeout"))

		repo := NewUserRepository(db)
		user, err := repo.FindByEmail(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "query user by email")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUserRepository_FindByID tests finding user by ID
func TestUserRepository_FindByID(t *testing.T) {
	t.Run("When find user by ID successfully", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()
		name := "User One"
		email := "user1@unsaac.edu.pe"
		role := domain.RoleStudent
		now := time.Now().UTC()

		// After the avatar-borders PR 2, find_by_id.sql returns 16
		// columns: the original 9 user columns + selected_border_id
		// (nullable) + the 6-column border row from the LEFT JOIN
		// (also nullable — the join only matches when the user has
		// a selected border).
		rows := sqlmock.NewRows([]string{
			"id", "name", "email", "role", "avatar_url", "avatar_version",
			"verified", "created_at", "blocked", "selected_border_id",
			"border_id", "border_slug", "border_name", "border_asset_url",
			"border_thumbnail_url", "border_tier",
		}).AddRow(
			userID, name, email, role, nil, 0, false, now, false,
			nil, nil, nil, nil, nil, nil, nil,
		)

		mock.ExpectQuery(queryFindUserByID).
			WithArgs(userID).
			WillReturnRows(rows)

		repo := NewUserRepository(db)
		user, err := repo.FindByID(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, name, user.Name)
		assert.Equal(t, email, user.Email)
		assert.Nil(t, user.SelectedBorderID, "no border in the row → no SelectedBorderID")
		assert.Nil(t, user.SelectedBorder, "no border in the row → no SelectedBorder")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When find user by ID with selected border hydrates SelectedBorder", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()
		borderID := uuid.New()
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{
			"id", "name", "email", "role", "avatar_url", "avatar_version",
			"verified", "created_at", "blocked", "selected_border_id",
			"border_id", "border_slug", "border_name", "border_asset_url",
			"border_thumbnail_url", "border_tier",
		}).AddRow(
			userID, "User", "u@u.com", domain.RoleStudent, nil, 0, false, now, false,
			borderID.String(), borderID.String(), "condor", "Cóndor",
			"https://cdn/condor.png", "https://cdn/condor_t.png", "free",
		)

		mock.ExpectQuery(queryFindUserByID).
			WithArgs(userID).
			WillReturnRows(rows)

		repo := NewUserRepository(db)
		user, err := repo.FindByID(context.Background(), userID)
		assert.NoError(t, err)
		require.NotNil(t, user)
		require.NotNil(t, user.SelectedBorderID)
		assert.Equal(t, borderID, *user.SelectedBorderID)
		require.NotNil(t, user.SelectedBorder)
		assert.Equal(t, "condor", user.SelectedBorder.Slug)
		assert.Equal(t, "free", user.SelectedBorder.Tier)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When user by ID not found", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()

		mock.ExpectQuery(queryFindUserByID).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		repo := NewUserRepository(db)
		user, err := repo.FindByID(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.True(t, domain.IsNotFound(err))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When database error on FindByID", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		userID := uuid.New()

		mock.ExpectQuery(queryFindUserByID).
			WithArgs(userID).
			WillReturnError(errors.New("network error"))

		repo := NewUserRepository(db)
		user, err := repo.FindByID(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "query user by id")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUserRepository_MarkVerified tests marking user as verified
func TestUserRepository_MarkVerified(t *testing.T) {
	t.Run("When mark user verified successfully", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"

		mock.ExpectExec(queryMarkUserVerified).
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

		repo := NewUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When mark verified with user not found", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "nonexistent@unsaac.edu.pe"

		mock.ExpectExec(queryMarkUserVerified).
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		repo := NewUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.Error(t, err)
		assert.True(t, domain.IsNotFound(err))
		assert.Contains(t, err.Error(), "mark verified")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When mark verified fails with db error", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"

		mock.ExpectExec(queryMarkUserVerified).
			WithArgs(email).
			WillReturnError(errors.New("connection refused"))

		repo := NewUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update verified")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When rows affected check fails", func(t *testing.T) {
		db, mock := setupUserTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"

		// Using NewErrorResult to simulate an error when getting rows affected
		mock.ExpectExec(queryMarkUserVerified).
			WithArgs(email).
			WillReturnError(errors.New("rows affected error"))

		repo := NewUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update verified")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
