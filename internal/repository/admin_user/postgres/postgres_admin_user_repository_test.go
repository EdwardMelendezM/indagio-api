package admin_users_postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"indagio-api/internal/domain"
)

func setupAdminUserTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("unable to create mock db: %v", err)
	}
	return db, mock
}

func TestAdminUserRepository_FindByEmail(t *testing.T) {
	t.Run("When find admin user by email successfully", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		adminID := uuid.New()
		now := time.Now().UTC()
		email := "admin@unsaac.edu.pe"

		rows := sqlmock.NewRows([]string{"id", "email", "created_at"}).
			AddRow(adminID.String(), email, now)

		mock.ExpectQuery(queryFindByEmail).
			WithArgs(email).
			WillReturnRows(rows)

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindByEmail(context.Background(), email)

		assert.NoError(t, err)
		assert.NotNil(t, admin)
		assert.Equal(t, adminID, admin.ID)
		assert.Equal(t, email, admin.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When admin user not found by email", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		email := "nonexistent@unsaac.edu.pe"

		mock.ExpectQuery(queryFindByEmail).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindByEmail(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, admin)
		assert.True(t, domain.IsNotFound(err))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When find admin user by email fails with db error", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		email := "admin@unsaac.edu.pe"

		mock.ExpectQuery(queryFindByEmail).
			WithArgs(email).
			WillReturnError(errors.New("connection refused"))

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindByEmail(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, admin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdminUserRepository_FindById(t *testing.T) {
	t.Run("When find admin user by ID successfully", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		adminID := uuid.New()
		now := time.Now().UTC()
		email := "admin@unsaac.edu.pe"

		rows := sqlmock.NewRows([]string{"id", "email", "created_at"}).
			AddRow(adminID.String(), email, now)

		mock.ExpectQuery(queryFindById).
			WithArgs(adminID.String()).
			WillReturnRows(rows)

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindById(context.Background(), adminID)

		assert.NoError(t, err)
		assert.NotNil(t, admin)
		assert.Equal(t, adminID, admin.ID)
		assert.Equal(t, email, admin.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When admin user not found by ID", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		adminID := uuid.New()

		mock.ExpectQuery(queryFindById).
			WithArgs(adminID.String()).
			WillReturnError(sql.ErrNoRows)

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindById(context.Background(), adminID)

		assert.Error(t, err)
		assert.Nil(t, admin)
		assert.True(t, domain.IsNotFound(err))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When find admin user by ID fails with db error", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		adminID := uuid.New()

		mock.ExpectQuery(queryFindById).
			WithArgs(adminID.String()).
			WillReturnError(errors.New("connection timeout"))

		repo := NewAdminUserRepository(db)
		admin, err := repo.FindById(context.Background(), adminID)

		assert.Error(t, err)
		assert.Nil(t, admin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdminUserRepository_MarkVerified(t *testing.T) {
	t.Run("When mark admin user verified successfully", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		email := "admin@unsaac.edu.pe"

		mock.ExpectExec(queryMarkVerified).
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(0, 1))

		repo := NewAdminUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When mark verified fails with db error", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		email := "admin@unsaac.edu.pe"

		mock.ExpectExec(queryMarkVerified).
			WithArgs(email).
			WillReturnError(errors.New("connection refused"))

		repo := NewAdminUserRepository(db)
		err := repo.MarkVerified(context.Background(), email)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdminUserRepository_Create(t *testing.T) {
	t.Run("When create admin user successfully", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		adminID := uuid.New()
		now := time.Now().UTC()
		email := "newadmin@unsaac.edu.pe"

		rows := sqlmock.NewRows([]string{"id", "created_at"}).
			AddRow(adminID.String(), now)

		mock.ExpectQuery(queryCreate).
			WithArgs(email).
			WillReturnRows(rows)

		repo := NewAdminUserRepository(db)
		admin, err := repo.Create(context.Background(), email)

		assert.NoError(t, err)
		assert.NotNil(t, admin)
		assert.Equal(t, adminID, admin.ID)
		assert.Equal(t, email, admin.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When create admin user fails with db error", func(t *testing.T) {
		db, mock := setupAdminUserTest(t)
		defer db.Close()

		email := "newadmin@unsaac.edu.pe"

		mock.ExpectQuery(queryCreate).
			WithArgs(email).
			WillReturnError(errors.New("unique constraint violation"))

		repo := NewAdminUserRepository(db)
		admin, err := repo.Create(context.Background(), email)

		assert.Error(t, err)
		assert.Nil(t, admin)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
