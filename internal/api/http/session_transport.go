package http

import (
	"time"

	"github.com/endge-lab/service-template-go/internal/usecase"
)

type SessionResponse struct {
	Session *SessionInfoResponse `json:"session,omitempty"`
	User    *UserResponse        `json:"user,omitempty"`
}

// SessionInfoResponse contains current JWT/session metadata returned to the caller.
type SessionInfoResponse struct {
	ID        string   `json:"id,omitempty"`
	SessionID string   `json:"sessionId,omitempty"`
	App       string   `json:"app,omitempty"`
	Platform  string   `json:"platform,omitempty"`
	Scope     []string `json:"scope,omitempty"`
	ExpiresAt string   `json:"expiresAt,omitempty"`
}

type UserResponse struct {
	ID          string    `json:"id"`
	AuthUserID  string    `json:"authUserId"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Role        string    `json:"role,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newSessionResponse(output *usecase.LoadSessionOutput) *SessionResponse {
	if output == nil {
		return nil
	}

	response := &SessionResponse{}

	if output.Session != nil {
		response.Session = &SessionInfoResponse{
			ID:        output.Session.ID,
			SessionID: output.Session.SessionID,
			App:       output.Session.App,
			Platform:  output.Session.Platform,
			Scope:     output.Session.Scope,
			ExpiresAt: output.Session.ExpiresAt,
		}
	}

	if output.User != nil {
		response.User = &UserResponse{
			ID:          output.User.ID,
			AuthUserID:  output.User.AuthUserID,
			Username:    output.User.Username,
			DisplayName: output.User.DisplayName,
			Role:        output.User.Role,
			CreatedAt:   output.User.CreatedAt,
			UpdatedAt:   output.User.UpdatedAt,
		}
	}

	return response
}
