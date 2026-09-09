package entities

import (
	"encoding/json"
	"time"
)

// BridgeMessage — закрытый protocol v1 envelope, не arbitrary application RPC.
type BridgeMessage struct {
	Type         string          `json:"type"`
	ID           string          `json:"id,omitempty"`
	SessionID    string          `json:"sessionId,omitempty"`
	TargetID     string          `json:"targetId,omitempty"`
	Identity     string          `json:"identity,omitempty"`
	ExpectedHash string          `json:"expectedHash,omitempty"`
	Accepted     bool            `json:"accepted,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
	Error        string          `json:"error,omitempty"`
}

type BridgeHello struct {
	Protocol          int    `json:"protocol"`
	WorkspaceIdentity string `json:"workspaceIdentity"`
	Label             string `json:"label"`
	Debug             bool   `json:"debug"`
}

// BridgePrincipal создаётся только после HTTP authentication, не из hello.
type BridgePrincipal struct {
	UserID    string
	SessionID string
	ExpiresAt time.Time
}

type BridgeConfigurator struct {
	InstanceID  string `json:"instanceId"`
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Label       string `json:"label"`
}

type BridgeClient struct {
	InstanceID string `json:"instanceId"`
	Label      string `json:"label"`
}

type BridgeSession struct {
	SessionID      string `json:"sessionId"`
	ClientID       string `json:"clientId"`
	ConfiguratorID string `json:"configuratorId"`
}
