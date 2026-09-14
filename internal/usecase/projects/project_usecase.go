package projects

import (
	"context"
	"fmt"
	"strings"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type projectUsecase struct {
	repo domain.ProjectRepository
}

func NewProjectUsecase(repo domain.ProjectRepository) domain.ProjectUsecase {
	return &projectUsecase{repo: repo}
}

func (uc *projectUsecase) CreateProject(ctx context.Context, ownerID uuid.UUID, name, description string) (*domain.Project, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if ownerID == uuid.Nil {
		return nil, fmt.Errorf("owner id: %w", domain.ErrValidation)
	}
	if len(name) < 2 || len(name) > 160 {
		return nil, fmt.Errorf("project name: %w", domain.ErrValidation)
	}

	project, err := uc.repo.Create(ctx, ownerID, name, description)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	if _, err = uc.repo.AddMember(ctx, project.ID, ownerID, ownerID, domain.ProjectMemberRoleOwner); err != nil {
		return nil, fmt.Errorf("add owner membership: %w", err)
	}

	return project, nil
}

func (uc *projectUsecase) GetProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*domain.Project, error) {
	if userID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("invalid project access: %w", domain.ErrValidation)
	}

	project, err := uc.repo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	for _, member := range members {
		if member.UserID == userID {
			return project, nil
		}
	}

	return nil, fmt.Errorf("project access: %w", domain.ErrForbidden)
}

func (uc *projectUsecase) ListProjects(ctx context.Context, userID uuid.UUID, status *domain.ProjectStatus) ([]domain.Project, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user id: %w", domain.ErrValidation)
	}
	projects, err := uc.repo.ListByUser(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (uc *projectUsecase) UpdateProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, name *string, description *string, status *domain.ProjectStatus) (*domain.Project, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	if name != nil {
		*name = strings.TrimSpace(*name)
		if len(*name) < 2 || len(*name) > 160 {
			return nil, fmt.Errorf("project name: %w", domain.ErrValidation)
		}
	}
	if description != nil {
		*description = strings.TrimSpace(*description)
	}

	_, err := uc.repo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	allowed := false
	for _, member := range members {
		if member.UserID == actorID && (member.Role == domain.ProjectMemberRoleOwner || member.Role == domain.ProjectMemberRoleAdmin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("project update: %w", domain.ErrForbidden)
	}

	project, err := uc.repo.Update(ctx, projectID, name, description, status)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return project, nil
}

func (uc *projectUsecase) ArchiveProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) error {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	allowed := false
	for _, member := range members {
		if member.UserID == actorID && (member.Role == domain.ProjectMemberRoleOwner || member.Role == domain.ProjectMemberRoleAdmin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("archive project: %w", domain.ErrForbidden)
	}
	return uc.repo.Archive(ctx, projectID)
}

func (uc *projectUsecase) DeleteProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) error {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list members: %w", err)
	}
	allowed := false
	for _, member := range members {
		if member.UserID == actorID && member.Role == domain.ProjectMemberRoleOwner {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("delete project: %w", domain.ErrForbidden)
	}
	return uc.repo.Delete(ctx, projectID)
}

func (uc *projectUsecase) AddMember(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, userID uuid.UUID, role domain.ProjectMemberRole) (*domain.ProjectMember, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("actor/project/user id: %w", domain.ErrValidation)
	}
	if role != domain.ProjectMemberRoleAdmin && role != domain.ProjectMemberRoleMember {
		return nil, fmt.Errorf("role: %w", domain.ErrValidation)
	}

	if _, err := uc.repo.GetByID(ctx, projectID); err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	allowed := false
	for _, member := range members {
		if member.UserID == actorID && (member.Role == domain.ProjectMemberRoleOwner || member.Role == domain.ProjectMemberRoleAdmin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("add member: %w", domain.ErrForbidden)
	}

	for _, member := range members {
		if member.UserID == userID {
			return nil, fmt.Errorf("member already exists: %w", domain.ErrConflict)
		}
	}

	return uc.repo.AddMember(ctx, projectID, userID, actorID, role)
}

func (uc *projectUsecase) InviteMember(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, email string, role domain.ProjectMemberRole, expiresAt time.Time) (*domain.ProjectInvitation, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	if email == "" {
		return nil, fmt.Errorf("email: %w", domain.ErrValidation)
	}
	if role != domain.ProjectMemberRoleAdmin && role != domain.ProjectMemberRoleMember {
		return nil, fmt.Errorf("role: %w", domain.ErrValidation)
	}
	if expiresAt.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("invitation expiry: %w", domain.ErrValidation)
	}

	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	allowed := false
	for _, member := range members {
		if member.UserID == actorID && (member.Role == domain.ProjectMemberRoleOwner || member.Role == domain.ProjectMemberRoleAdmin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("invite member: %w", domain.ErrForbidden)
	}

	return uc.repo.CreateInvitation(ctx, projectID, actorID, email, role, expiresAt)
}

func (uc *projectUsecase) ListMembers(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	if actorID == uuid.Nil || projectID == uuid.Nil {
		return nil, fmt.Errorf("actor/project id: %w", domain.ErrValidation)
	}
	members, err := uc.repo.ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	for _, member := range members {
		if member.UserID == actorID {
			return members, nil
		}
	}
	return nil, fmt.Errorf("project access: %w", domain.ErrForbidden)
}
