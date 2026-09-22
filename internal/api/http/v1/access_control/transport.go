package access_control

import (
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type UserResponse struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Active      bool   `json:"active"`
}

type GrantResponse struct {
	ID                   string         `json:"id"`
	User                 UserResponse   `json:"user"`
	ScopeType            string         `json:"scopeType"`
	WorkspaceIdentity    *string        `json:"workspaceIdentity,omitempty"`
	WorkspaceDisplayName *string        `json:"workspaceDisplayName,omitempty"`
	Role                 string         `json:"role"`
	CreatedBy            entities.Actor `json:"createdBy"`
	UpdatedBy            entities.Actor `json:"updatedBy"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type UserListResponse struct {
	Items      []UserResponse `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

type GrantListResponse struct {
	Items      []GrantResponse `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

type PutRequest struct {
	UserID            string `json:"userId" validate:"required,uuid"`
	ScopeType         string `json:"scopeType" validate:"required,oneof=platform workspace"`
	WorkspaceIdentity string `json:"workspaceIdentity,omitempty" validate:"omitempty,max=160"`
	Role              string `json:"role" validate:"required,oneof=viewer editor admin"`
}

type BulkSelectionRequest struct {
	Type                string   `json:"type" validate:"required,oneof=all-active selected"`
	WorkspaceIdentities []string `json:"workspaceIdentities,omitempty" validate:"omitempty,dive,required,max=160"`
}

type BulkRequest struct {
	UserID    string               `json:"userId" validate:"required,uuid"`
	Role      string               `json:"role" validate:"required,oneof=viewer editor admin"`
	Selection BulkSelectionRequest `json:"selection" validate:"required"`
}

type BulkResponse struct {
	Affected int `json:"affected"`
	Created  int `json:"created"`
	Updated  int `json:"updated"`
}

func newUserResponse(value entities.AccessGrantUser) UserResponse {
	return UserResponse{ID: value.ID, ProviderID: value.ProviderID, Username: value.Username, DisplayName: value.DisplayName, Active: value.Active}
}

func newGrantResponse(value entities.AccessGrant) GrantResponse {
	return GrantResponse{
		ID: value.ID, User: newUserResponse(value.User), ScopeType: value.ScopeType,
		WorkspaceIdentity: value.WorkspaceIdentity, WorkspaceDisplayName: value.WorkspaceDisplayName,
		Role: value.Role, CreatedBy: value.CreatedBy, UpdatedBy: value.UpdatedBy,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}
