package entities

import "time"

// MappedAccessGrant is an assignment derived from verified provider attributes.
type MappedAccessGrant struct {
	Scope     string `json:"scope"`
	Workspace string `json:"workspace,omitempty"`
	Role      string `json:"role"`
}

// ExternalAccessSnapshot contains no raw JWT or provider claims.
type ExternalAccessSnapshot struct {
	ConfigVersion string              `json:"configVersion"`
	TokenHash     string              `json:"tokenHash"`
	IssuedAt      time.Time           `json:"issuedAt"`
	ExpiresAt     time.Time           `json:"expiresAt"`
	Grants        []MappedAccessGrant `json:"grants"`
}

// ExternalAccessState is the per-user watermark persisted with the assignments.
type ExternalAccessState struct {
	ConfigVersion      string
	TokenHash          string
	IssuedAt           time.Time
	GrantsHash         string
	LastSynchronizedAt time.Time
}

type AccessManagement struct {
	Mode               string
	SourceName         string
	LastSynchronizedAt *time.Time
}
