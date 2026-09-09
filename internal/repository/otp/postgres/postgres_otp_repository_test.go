package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupOTPTest(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("unable to create mock db: %v", err)
	}
	return db, mock
}

// TestOTPRepository_Create tests the OTP creation functionality
func TestOTPRepository_Create(t *testing.T) {
	t.Run("When create OTP successfully", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"

		// Expect the insert query to succeed
		mock.ExpectExec(`INSERT INTO otps (email, code, expires_at, used)
VALUES ($1, $2, $3, false)`).
			WithArgs(email, code, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewOTPRepository(db)
		err := repo.Create(context.Background(), email, code)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When create OTP fails with db error", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"

		// Expect the insert to fail
		mock.ExpectExec(`INSERT INTO otps (email, code, expires_at, used)
VALUES ($1, $2, $3, false)`).
			WithArgs(email, code, sqlmock.AnyArg()).
			WillReturnError(errors.New("unique constraint violation"))

		repo := NewOTPRepository(db)
		err := repo.Create(context.Background(), email, code)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insert otp")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestOTPRepository_ValidateAndConsume tests the OTP validation and consumption
func TestOTPRepository_ValidateAndConsume(t *testing.T) {
	t.Run("When validate and consume OTP successfully", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"
		expiresAt := time.Now().Add(10 * time.Minute).UTC()

		// Mock the select query that fetches the OTP
		rows := sqlmock.NewRows([]string{"code", "expires_at"}).
			AddRow(code, expiresAt)

		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnRows(rows)

		// Mock the consume update
		mock.ExpectExec(`UPDATE otps
SET used = true
WHERE email = $1 AND used = false`).
			WithArgs(email).
			WillReturnResult(sqlmock.NewResult(1, 1))

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, code)

		assert.NoError(t, err)
		assert.True(t, valid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When OTP not found", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"

		// Mock the select query that returns no rows
		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, code)

		assert.NoError(t, err)
		assert.False(t, valid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When OTP is expired", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"
		expiresAt := time.Now().Add(-5 * time.Minute).UTC() // Already expired

		rows := sqlmock.NewRows([]string{"code", "expires_at"}).
			AddRow(code, expiresAt)

		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnRows(rows)

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, code)

		assert.NoError(t, err)
		assert.False(t, valid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When OTP code does not match", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		correctCode := "123456"
		incorrectCode := "654321"
		expiresAt := time.Now().Add(10 * time.Minute).UTC()

		rows := sqlmock.NewRows([]string{"code", "expires_at"}).
			AddRow(correctCode, expiresAt)

		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnRows(rows)

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, incorrectCode)

		assert.NoError(t, err)
		assert.False(t, valid)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When database query error occurs", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"

		// Mock the select query to fail with a generic database error
		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnError(errors.New("connection timeout"))

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, code)

		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "query otp")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("When consume OTP fails", func(t *testing.T) {
		db, mock := setupOTPTest(t)
		defer db.Close()

		email := "test@unsaac.edu.pe"
		code := "123456"
		expiresAt := time.Now().Add(10 * time.Minute).UTC()

		rows := sqlmock.NewRows([]string{"code", "expires_at"}).
			AddRow(code, expiresAt)

		mock.ExpectQuery(`SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1`).
			WithArgs(email).
			WillReturnRows(rows)

		// Mock the consume to fail
		mock.ExpectExec(`UPDATE otps
SET used = true
WHERE email = $1 AND used = false`).
			WithArgs(email).
			WillReturnError(errors.New("update failed"))

		repo := NewOTPRepository(db)
		valid, err := repo.ValidateAndConsume(context.Background(), email, code)

		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "consume otp")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
