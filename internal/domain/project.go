package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProjectStatus defines the lifecycle of a project within the platform.
type ProjectStatus string

const (
	ProjectStatusDraft    ProjectStatus = "draft"
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusPaused   ProjectStatus = "paused"
	ProjectStatusClosed   ProjectStatus = "closed"
	ProjectStatusArchived ProjectStatus = "archived"
)

// ProjectMemberRole is the access role a user has inside a project.
type ProjectMemberRole string

const (
	ProjectMemberRoleOwner  ProjectMemberRole = "owner"
	ProjectMemberRoleAdmin  ProjectMemberRole = "admin"
	ProjectMemberRoleMember ProjectMemberRole = "member"
)

// Project represents a research project owned by a user and shared with members.
type Project struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        string
	Description string
	Status      ProjectStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
	DeletedAt   *time.Time
}

// ProjectMember is a direct membership assignment linking a user to a project.
type ProjectMember struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	UserID        uuid.UUID
	UserEmail     string
	UserName      string
	UserRole      string
	UserCreatedAt time.Time
	UserUpdatedAt time.Time
	Role          ProjectMemberRole
	Status        string
	InvitedBy     *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProjectInvitation is an invitation sent to a user email for a project.
type ProjectInvitation struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Email     string
	Role      ProjectMemberRole
	InvitedBy uuid.UUID
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProjectRepository defines the database contract for project management.
type ProjectRepository interface {
	Create(ctx context.Context, ownerID uuid.UUID, name, description string) (*Project, error)
	GetByID(ctx context.Context, projectID uuid.UUID) (*Project, error)
	ListByUser(ctx context.Context, userID uuid.UUID, status *ProjectStatus) ([]Project, error)
	Update(ctx context.Context, projectID uuid.UUID, name *string, description *string, status *ProjectStatus) (*Project, error)
	Archive(ctx context.Context, projectID uuid.UUID) error
	Delete(ctx context.Context, projectID uuid.UUID) error
	AddMember(ctx context.Context, projectID, userID, invitedBy uuid.UUID, role ProjectMemberRole) (*ProjectMember, error)
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]ProjectMember, error)
	CreateInvitation(ctx context.Context, projectID, invitedBy uuid.UUID, email string, role ProjectMemberRole, expiresAt time.Time) (*ProjectInvitation, error)
	ListInvitations(ctx context.Context, projectID uuid.UUID) ([]ProjectInvitation, error)
	AcceptInvitation(ctx context.Context, invitationID uuid.UUID, userID uuid.UUID) (*ProjectMember, error)
}

// ProjectUsecase handles project-level business rules and access checks.
type ProjectUsecase interface {
	CreateProject(ctx context.Context, ownerID uuid.UUID, name, description string) (*Project, error)
	GetProject(ctx context.Context, userID uuid.UUID, projectID uuid.UUID) (*Project, error)
	ListProjects(ctx context.Context, userID uuid.UUID, status *ProjectStatus) ([]Project, error)
	UpdateProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, name *string, description *string, status *ProjectStatus) (*Project, error)
	ArchiveProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) error
	DeleteProject(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) error
	AddMember(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, userID uuid.UUID, role ProjectMemberRole) (*ProjectMember, error)
	InviteMember(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID, email string, role ProjectMemberRole, expiresAt time.Time) (*ProjectInvitation, error)
	ListMembers(ctx context.Context, actorID uuid.UUID, projectID uuid.UUID) ([]ProjectMember, error)
}
