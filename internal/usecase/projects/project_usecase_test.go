package projects

import (
	"context"
	"errors"
	"testing"
	"time"

	"indagio-api/internal/domain"

	"github.com/google/uuid"
)

type stubProjectRepo struct {
	projects map[uuid.UUID]*domain.Project
	members  map[uuid.UUID][]domain.ProjectMember
}

func (s *stubProjectRepo) Create(ctx context.Context, ownerID uuid.UUID, name, description string) (*domain.Project, error) {
	project := &domain.Project{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		Status:      domain.ProjectStatusDraft,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	s.projects[project.ID] = project
	return project, nil
}

func (s *stubProjectRepo) GetByID(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	p, ok := s.projects[projectID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *stubProjectRepo) ListByUser(ctx context.Context, userID uuid.UUID, status *domain.ProjectStatus) ([]domain.Project, error) {
	out := make([]domain.Project, 0)
	for _, project := range s.projects {
		if project.OwnerID == userID {
			if status != nil && project.Status != *status {
				continue
			}
			out = append(out, *project)
		}
	}
	return out, nil
}

func (s *stubProjectRepo) Update(ctx context.Context, projectID uuid.UUID, name *string, description *string, status *domain.ProjectStatus) (*domain.Project, error) {
	project, ok := s.projects[projectID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if name != nil {
		project.Name = *name
	}
	if description != nil {
		project.Description = *description
	}
	if status != nil {
		project.Status = *status
	}
	project.UpdatedAt = time.Now().UTC()
	return project, nil
}

func (s *stubProjectRepo) Archive(ctx context.Context, projectID uuid.UUID) error {
	project, ok := s.projects[projectID]
	if !ok {
		return domain.ErrNotFound
	}
	project.Status = domain.ProjectStatusArchived
	project.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *stubProjectRepo) Delete(ctx context.Context, projectID uuid.UUID) error {
	if _, ok := s.projects[projectID]; !ok {
		return domain.ErrNotFound
	}
	delete(s.projects, projectID)
	return nil
}

func (s *stubProjectRepo) AddMember(ctx context.Context, projectID, userID, invitedBy uuid.UUID, role domain.ProjectMemberRole) (*domain.ProjectMember, error) {
	member := &domain.ProjectMember{
		ID:        uuid.New(),
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		Status:    "active",
		InvitedBy: &invitedBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.members[projectID] = append(s.members[projectID], *member)
	return member, nil
}

func (s *stubProjectRepo) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	list := s.members[projectID]
	filtered := list[:0]
	for _, member := range list {
		if member.UserID != userID {
			filtered = append(filtered, member)
		}
	}
	s.members[projectID] = filtered
	return nil
}

func (s *stubProjectRepo) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	return s.members[projectID], nil
}

func (s *stubProjectRepo) CreateInvitation(ctx context.Context, projectID, invitedBy uuid.UUID, email string, role domain.ProjectMemberRole, expiresAt time.Time) (*domain.ProjectInvitation, error) {
	invitation := &domain.ProjectInvitation{
		ID:        uuid.New(),
		ProjectID: projectID,
		Email:     email,
		Role:      role,
		InvitedBy: invitedBy,
		Status:    "pending",
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	return invitation, nil
}

func (s *stubProjectRepo) ListInvitations(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectInvitation, error) {
	return nil, nil
}

func (s *stubProjectRepo) AcceptInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) (*domain.ProjectMember, error) {
	return nil, errors.New("not implemented")
}

func TestProjectUsecase_CreateProject_AddsOwnerMembership(t *testing.T) {
	ownerID := uuid.New()
	repo := &stubProjectRepo{projects: map[uuid.UUID]*domain.Project{}, members: map[uuid.UUID][]domain.ProjectMember{}}
	uc := NewProjectUsecase(repo)

	project, err := uc.CreateProject(context.Background(), ownerID, "Alpha Project", "Research project")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	if project.Name != "Alpha Project" {
		t.Fatalf("unexpected project name: %s", project.Name)
	}
	if len(repo.members[project.ID]) != 1 {
		t.Fatalf("expected owner membership to be created, got %d", len(repo.members[project.ID]))
	}
	if repo.members[project.ID][0].Role != domain.ProjectMemberRoleOwner {
		t.Fatalf("expected owner role, got %s", repo.members[project.ID][0].Role)
	}
}

func TestProjectUsecase_AddMember_Success(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	projectID := uuid.New()
	repo := &stubProjectRepo{
		projects: map[uuid.UUID]*domain.Project{projectID: &domain.Project{ID: projectID, OwnerID: ownerID, Name: "Alpha Project", Status: domain.ProjectStatusDraft}},
		members:  map[uuid.UUID][]domain.ProjectMember{projectID: {{UserID: ownerID, Role: domain.ProjectMemberRoleOwner}}},
	}
	uc := NewProjectUsecase(repo)

	member, err := uc.AddMember(context.Background(), ownerID, projectID, targetID, domain.ProjectMemberRoleMember)
	if err != nil {
		t.Fatalf("AddMember returned error: %v", err)
	}
	if member.UserID != targetID {
		t.Fatalf("expected target user %s, got %s", targetID, member.UserID)
	}
	if len(repo.members[projectID]) != 2 {
		t.Fatalf("expected 2 members after add, got %d", len(repo.members[projectID]))
	}
}

func TestProjectUsecase_AddMember_ValidatesRole(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	projectID := uuid.New()
	repo := &stubProjectRepo{
		projects: map[uuid.UUID]*domain.Project{projectID: &domain.Project{ID: projectID, OwnerID: ownerID, Name: "Alpha Project", Status: domain.ProjectStatusDraft}},
		members:  map[uuid.UUID][]domain.ProjectMember{projectID: {{UserID: ownerID, Role: domain.ProjectMemberRoleOwner}}},
	}
	uc := NewProjectUsecase(repo)

	_, err := uc.AddMember(context.Background(), ownerID, projectID, targetID, "invalid")
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestProjectUsecase_AddMember_RejectsUnauthorizedActor(t *testing.T) {
	ownerID := uuid.New()
	memberID := uuid.New()
	targetID := uuid.New()
	projectID := uuid.New()
	repo := &stubProjectRepo{
		projects: map[uuid.UUID]*domain.Project{projectID: &domain.Project{ID: projectID, OwnerID: ownerID, Name: "Alpha Project", Status: domain.ProjectStatusDraft}},
		members:  map[uuid.UUID][]domain.ProjectMember{projectID: {{UserID: memberID, Role: domain.ProjectMemberRoleMember}}},
	}
	uc := NewProjectUsecase(repo)

	_, err := uc.AddMember(context.Background(), memberID, projectID, targetID, domain.ProjectMemberRoleMember)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestProjectUsecase_AddMember_RejectsDuplicateMember(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	projectID := uuid.New()
	repo := &stubProjectRepo{
		projects: map[uuid.UUID]*domain.Project{projectID: &domain.Project{ID: projectID, OwnerID: ownerID, Name: "Alpha Project", Status: domain.ProjectStatusDraft}},
		members: map[uuid.UUID][]domain.ProjectMember{projectID: {
			{UserID: ownerID, Role: domain.ProjectMemberRoleOwner},
			{UserID: targetID, Role: domain.ProjectMemberRoleMember},
		}},
	}
	uc := NewProjectUsecase(repo)

	_, err := uc.AddMember(context.Background(), ownerID, projectID, targetID, domain.ProjectMemberRoleAdmin)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestProjectUsecase_InviteMember_ValidatesRole(t *testing.T) {
	ownerID := uuid.New()
	projectID := uuid.New()
	repo := &stubProjectRepo{projects: map[uuid.UUID]*domain.Project{}, members: map[uuid.UUID][]domain.ProjectMember{projectID: {{UserID: ownerID, Role: domain.ProjectMemberRoleOwner}}}}
	uc := NewProjectUsecase(repo)

	_, err := uc.InviteMember(context.Background(), ownerID, projectID, "alex@example.com", "invalid", time.Now().UTC().Add(time.Hour))
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
