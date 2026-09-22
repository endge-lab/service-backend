package entities

import (
	"encoding/json"
	"time"
)

type MockOwner struct {
	ActorID     string
	WorkspaceID string
}
type MockParameters struct {
	IntervalMs      int   `json:"intervalMs"`
	ItemsPerMessage int   `json:"itemsPerMessage"`
	EmitImmediately *bool `json:"emitImmediately,omitempty"`
}
type MockStream struct {
	ID                string         `json:"id"`
	Status            string         `json:"status"`
	Seed              string         `json:"seed"`
	Profile           string         `json:"profile"`
	Parameters        MockParameters `json:"parameters"`
	ParametersVersion uint64         `json:"parametersVersion"`
	Sequence          uint64         `json:"sequence"`
	IdleTimeoutMs     int64          `json:"idleTimeoutMs"`
	ExpiresAt         time.Time      `json:"expiresAt"`
	EventsURL         string         `json:"eventsUrl,omitempty"`
}
type MockEvent struct {
	Type              string            `json:"type"`
	StreamID          string            `json:"streamId"`
	Sequence          uint64            `json:"sequence,omitempty"`
	ParametersVersion uint64            `json:"parametersVersion,omitempty"`
	Items             []json.RawMessage `json:"items,omitempty"`
	ErrorCode         string            `json:"errorCode,omitempty"`
	ErrorMessage      string            `json:"errorMessage,omitempty"`
}
