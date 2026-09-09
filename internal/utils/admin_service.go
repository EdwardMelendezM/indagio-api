package utils

import (
	"database/sql"
	"foro-unsaac-backend/internal/domain"
)

type adminService struct {
	db *sql.DB
}

func NewAdminService(db *sql.DB) domain.AdminService {
	return &adminService{db: db}
}

func (a adminService) Verify(id string) bool {
	err := a.db.QueryRow("SELECT id FROM user_adm WHERE id = $1 AND deleted_at IS NULL", id).Scan(&id)
	if err != nil {
		return false
	}
	return true
}
