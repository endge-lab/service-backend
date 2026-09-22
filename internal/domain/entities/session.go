package entities

import "time"

type User struct {
	ID          string    `json:"id"`
	ProviderID  string    `json:"providerId"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	AuthUserID  string    `json:"-"`
	Username    string    `json:"username,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Role        string    `json:"role,omitempty"`
	Active      bool      `json:"active"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SessionInfo struct {
	ID        string
	SessionID string
	App       string
	Platform  string
	Scope     []string
	ExpiresAt string
}
