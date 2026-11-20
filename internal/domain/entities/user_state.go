package entities

import (
	"encoding/json"
	"time"
)

type UserStateScope struct {
	WorkspaceID string
	Environment string
	Facets      string
}

type UserStateEntry struct {
	Key        string          `json:"key"`
	Value      json.RawMessage `json:"value" swaggertype:"object" extensions:"x-endge-json-value"`
	Deleted    bool            `json:"deleted"`
	Revision   int64           `json:"revision"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	UpdatedBy  string          `json:"updatedBy,omitempty"`
	MutationID string          `json:"-"`
}

type UserStateUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Count       int    `json:"count"`
}
