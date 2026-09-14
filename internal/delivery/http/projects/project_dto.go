package projects

import (
	"time"

	"indagio-api/internal/domain"
)

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=160"`
	Description string `json:"description"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

type InviteMemberRequest struct {
	Email     string    `json:"email" binding:"required,email"`
	Role      string    `json:"role" binding:"required"`
	ExpiresAt time.Time `json:"expires_at" binding:"required"`
}

type ProjectResponse struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectMemberResponse struct {
	ID        string             `json:"id"`
	ProjectID string             `json:"project_id"`
	User      MemberUserResponse `json:"user"`
	Role      string             `json:"role"`
	Status    string             `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type MemberUserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectInvitationResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func ToProjectResponse(p *domain.Project) ProjectResponse {
	return ProjectResponse{
		ID:          p.ID.String(),
		OwnerID:     p.OwnerID.String(),
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func ToProjectMemberResponse(m domain.ProjectMember) ProjectMemberResponse {
	return ProjectMemberResponse{
		ID:        m.ID.String(),
		ProjectID: m.ProjectID.String(),
		User: MemberUserResponse{
			ID:        m.UserID.String(),
			Email:     m.UserEmail,
			Name:      m.UserName,
			Role:      string(m.UserRole),
			CreatedAt: m.UserCreatedAt,
			UpdatedAt: m.UserUpdatedAt,
		},
		Role:      string(m.Role),
		Status:    m.Status,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToProjectInvitationResponse(i *domain.ProjectInvitation) ProjectInvitationResponse {
	return ProjectInvitationResponse{
		ID:        i.ID.String(),
		ProjectID: i.ProjectID.String(),
		Email:     i.Email,
		Role:      string(i.Role),
		Status:    i.Status,
		ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt,
	}
}
