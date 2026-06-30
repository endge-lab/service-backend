package entities

import "time"

type User struct {
	ID          string
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SessionInfo struct {
	ID        string
	SessionID string
	App       string
	Platform  string
	Scope     []string
	ExpiresAt string
}
