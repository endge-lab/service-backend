package entities

import (
	"encoding/json"
	"time"
)

type Workspace struct {
	ID            string          `json:"id"`
	Identity      string          `json:"identity"`
	DisplayName   string          `json:"displayName"`
	Description   *string         `json:"description,omitempty"`
	DataMode      string          `json:"dataMode"`
	Configuration json.RawMessage `json:"configuration"`
	Meta          json.RawMessage `json:"meta"`
	Active        bool            `json:"active"`
	Generation    string          `json:"generation"`
	HeadSequence  int64           `json:"headSequence"`
	Revision      int             `json:"revision"`
	CreatedBy     Actor           `json:"createdBy"`
	UpdatedBy     Actor           `json:"updatedBy"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}
