package db

import (
	"database/sql"

	_ "github.com/lib/pq"

	"indagio-api/internal/config"
)

// Connect opens and validates a PostgreSQL connection using the provided config.
func Connect(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}
