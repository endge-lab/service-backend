package entities

import "time"

type AccessGrantUser struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Active      bool   `json:"active"`
}

type AccessGrant struct {
	ID                   string          `json:"id"`
	User                 AccessGrantUser `json:"user"`
	ScopeType            string          `json:"scopeType"`
	WorkspaceID          *string         `json:"workspaceId,omitempty"`
	WorkspaceIdentity    *string         `json:"workspaceIdentity,omitempty"`
	WorkspaceDisplayName *string         `json:"workspaceDisplayName,omitempty"`
	Role                 string          `json:"role"`
	CreatedBy            Actor           `json:"createdBy"`
	UpdatedBy            Actor           `json:"updatedBy"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

type AccessGrantCursor struct {
	Username string
	ID       string
}

type AccessGrantPage struct {
	Items []AccessGrant
	Next  *AccessGrantCursor
}

type ServiceUserPage struct {
	Items []AccessGrantUser
	Next  *AccessGrantCursor
}
