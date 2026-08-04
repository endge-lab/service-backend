package entities

import (
	"encoding/json"
	"time"
)

type Integration struct {
	ID          string          `json:"id"`
	Identity    string          `json:"identity"`
	DisplayName string          `json:"displayName"`
	Description *string         `json:"description,omitempty"`
	Version     string          `json:"version"`
	ManagedBy   string          `json:"managedBy"`
	ManagedByID *string         `json:"managedById,omitempty"`
	Meta        json.RawMessage `json:"meta"`
	Active      bool            `json:"active"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty"`
	Revision    int             `json:"revision"`
	CreatedBy   Actor           `json:"createdBy"`
	UpdatedBy   Actor           `json:"updatedBy"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}
