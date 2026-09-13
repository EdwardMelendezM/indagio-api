package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type postgresProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) domain.ProjectRepository {
	return &postgresProjectRepository{db: db}
}

func (r *postgresProjectRepository) Create(ctx context.Context, ownerID uuid.UUID, name, description string) (*domain.Project, error) {
	projectID := uuid.New()
	project := &domain.Project{
		ID:          projectID,
		OwnerID:     ownerID,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Status:      domain.ProjectStatusDraft,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, owner_id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, project.ID, project.OwnerID, project.Name, project.Description, project.Status, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return project, nil
}

func (r *postgresProjectRepository) GetByID(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	var p domain.Project
	err := r.db.QueryRowContext(ctx, `
		SELECT id, owner_id, name, description, status, created_at, updated_at, archived_at, deleted_at
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`, projectID).Scan(
		&p.ID,
		&p.OwnerID,
		&p.Name,
		&p.Description,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ArchivedAt,
		&p.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("project %s: %w", projectID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query project: %w", err)
	}
	return &p, nil
}

func (r *postgresProjectRepository) ListByUser(ctx context.Context, userID uuid.UUID, status *domain.ProjectStatus) ([]domain.Project, error) {
	query := `
		SELECT p.id, p.owner_id, p.name, p.description, p.status, p.created_at, p.updated_at, p.archived_at, p.deleted_at
		FROM projects p
		JOIN project_members pm ON pm.project_id = p.id
		WHERE pm.user_id = $1 AND p.deleted_at IS NULL
	`
	args := []any{userID}
	if status != nil {
		query += " AND p.status = $2"
		args = append(args, string(*status))
	}
	query += " ORDER BY p.updated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(
			&p.ID,
			&p.OwnerID,
			&p.Name,
			&p.Description,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.ArchivedAt,
			&p.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}
	return projects, nil
}

func (r *postgresProjectRepository) Update(ctx context.Context, projectID uuid.UUID, name *string, description *string, status *domain.ProjectStatus) (*domain.Project, error) {
	parts := []string{"updated_at = NOW()"}
	args := []any{projectID}
	idx := 2

	if name != nil {
		parts = append(parts, fmt.Sprintf("name = $%d", idx))
		args = append(args, strings.TrimSpace(*name))
		idx++
	}
	if description != nil {
		parts = append(parts, fmt.Sprintf("description = $%d", idx))
		args = append(args, strings.TrimSpace(*description))
		idx++
	}
	if status != nil {
		parts = append(parts, fmt.Sprintf("status = $%d", idx))
		args = append(args, string(*status))
		idx++
	}

	if len(parts) == 1 {
		return r.GetByID(ctx, projectID)
	}

	query := fmt.Sprintf(`UPDATE projects SET %s WHERE id = $1 AND deleted_at IS NULL`, strings.Join(parts, ", "))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, fmt.Errorf("project %s: %w", projectID, domain.ErrNotFound)
	}
	return r.GetByID(ctx, projectID)
}

func (r *postgresProjectRepository) Archive(ctx context.Context, projectID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE projects
		SET status = $2, archived_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, projectID, domain.ProjectStatusArchived)
	if err != nil {
		return fmt.Errorf("archive project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project %s: %w", projectID, domain.ErrNotFound)
	}
	return nil
}

func (r *postgresProjectRepository) Delete(ctx context.Context, projectID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE projects
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, projectID)
	if err != nil {
		return fmt.Errorf("soft delete project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project %s: %w", projectID, domain.ErrNotFound)
	}
	return nil
}

func (r *postgresProjectRepository) AddMember(ctx context.Context, projectID, userID, invitedBy uuid.UUID, role domain.ProjectMemberRole) (*domain.ProjectMember, error) {
	memberID := uuid.New()
	member := &domain.ProjectMember{
		ID:        memberID,
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		Status:    "active",
		InvitedBy: &invitedBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_members (id, project_id, user_id, role, status, invited_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (project_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			invited_by = EXCLUDED.invited_by,
			updated_at = NOW()
	`, member.ID, member.ProjectID, member.UserID, member.Role, member.Status, member.InvitedBy, member.CreatedAt, member.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert project member: %w", err)
	}
	return member, nil
}

func (r *postgresProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID)
	if err != nil {
		return fmt.Errorf("remove project member: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("member %s on project %s: %w", userID, projectID, domain.ErrNotFound)
	}
	return nil
}

func (r *postgresProjectRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pm.id, pm.project_id, pm.user_id, pm.role, pm.status, pm.invited_by, pm.created_at, pm.updated_at,
		       u.email, u.name, u.role, u.created_at, u.updated_at
		FROM project_members pm
		JOIN users u ON pm.user_id = u.id
		WHERE pm.project_id = $1
		ORDER BY pm.created_at ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project members: %w", err)
	}
	defer rows.Close()

	members := make([]domain.ProjectMember, 0)
	for rows.Next() {
		var m domain.ProjectMember
		var invitedBy sql.NullString
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.Role, &m.Status, &invitedBy, &m.CreatedAt, &m.UpdatedAt,
			&m.UserEmail, &m.UserName, &m.UserRole, &m.UserCreatedAt, &m.UserUpdatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		if invitedBy.Valid {
			parsed, err := uuid.Parse(invitedBy.String)
			if err == nil {
				m.InvitedBy = &parsed
			}
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

func (r *postgresProjectRepository) CreateInvitation(ctx context.Context, projectID, invitedBy uuid.UUID, email string, role domain.ProjectMemberRole, expiresAt time.Time) (*domain.ProjectInvitation, error) {
	invitation := &domain.ProjectInvitation{
		ID:        uuid.New(),
		ProjectID: projectID,
		Email:     strings.TrimSpace(strings.ToLower(email)),
		Role:      role,
		InvitedBy: invitedBy,
		Status:    "pending",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_invitations (id, project_id, email, role, invited_by, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, invitation.ID, invitation.ProjectID, invitation.Email, invitation.Role, invitation.InvitedBy, invitation.Status, invitation.ExpiresAt, invitation.CreatedAt, invitation.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create project invitation: %w", err)
	}
	return invitation, nil
}

func (r *postgresProjectRepository) ListInvitations(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectInvitation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, email, role, invited_by, status, expires_at, created_at, updated_at
		FROM project_invitations
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	invitations := make([]domain.ProjectInvitation, 0)
	for rows.Next() {
		var i domain.ProjectInvitation
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.Email, &i.Role, &i.InvitedBy, &i.Status, &i.ExpiresAt, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		invitations = append(invitations, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invitations: %w", err)
	}
	return invitations, nil
}

func (r *postgresProjectRepository) AcceptInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) (*domain.ProjectMember, error) {
	var invitation domain.ProjectInvitation
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, email, role, invited_by, status, expires_at
		FROM project_invitations
		WHERE id = $1 AND status = 'pending' AND expires_at > NOW()
	`, invitationID).Scan(&invitation.ID, &invitation.ProjectID, &invitation.Email, &invitation.Role, &invitation.InvitedBy, &invitation.Status, &invitation.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("invitation %s: %w", invitationID, domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query invitation: %w", err)
	}

	member, err := r.AddMember(ctx, invitation.ProjectID, userID, invitation.InvitedBy, invitation.Role)
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE project_invitations SET status = 'accepted', updated_at = NOW() WHERE id = $1`, invitationID)
	if err != nil {
		return nil, fmt.Errorf("accept invitation: %w", err)
	}
	return member, nil
}
