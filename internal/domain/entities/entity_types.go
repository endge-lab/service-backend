package entities

import (
	"time"

	"github.com/google/uuid"
)

type Component struct {
	ID            uuid.UUID      `json:"id"`
	Identity      string         `json:"identity"`
	DisplayName   string         `json:"display_name"`
	ComponentType string         `json:"component_type"`
	InputFields   []any          `json:"input_fields"`
	JSXScript     *string        `json:"jsx_script,omitempty"`
	RowSize       *string        `json:"row_size,omitempty"`
	Bindings      map[string]any `json:"bindings"`
	Columns       []any          `json:"columns"`
	Schema        map[string]any `json:"schema"`
	FolderID      uuid.UUID      `json:"folder_id"`
	ProjectID     *uuid.UUID     `json:"project_id,omitempty"`
	Active        bool           `json:"active"`
	Inherited     bool           `json:"inherited"`
	DeletedAt     *time.Time     `json:"deleted_at,omitempty"`
	Author        *string        `json:"author,omitempty"`
	Meta          map[string]any `json:"meta"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (Component) FolderEntityType() FolderEntityType {
	return FolderEntityTypeComponents
}

type Converter struct {
	ID          uuid.UUID      `json:"id"`
	Identity    string         `json:"identity"`
	DisplayName string         `json:"display_name"`
	Description *string        `json:"description,omitempty"`
	IsSystem    bool           `json:"is_system"`
	FolderID    *uuid.UUID     `json:"folder_id,omitempty"`
	ProjectID   *uuid.UUID     `json:"project_id,omitempty"`
	Inherited   bool           `json:"inherited"`
	Meta        map[string]any `json:"meta"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (Converter) FolderEntityType() FolderEntityType {
	return FolderEntityTypeConverters
}

type Query struct {
	ID                   uuid.UUID      `json:"id"`
	Identity             string         `json:"identity"`
	DisplayName          string         `json:"display_name"`
	Type                 string         `json:"type"`
	Endpoint             *string        `json:"endpoint,omitempty"`
	Query                *string        `json:"query,omitempty"`
	SubField             *string        `json:"sub_field,omitempty"`
	Method               *string        `json:"method,omitempty"`
	Headers              map[string]any `json:"headers"`
	TimeoutMs            *int32         `json:"timeout_ms,omitempty"`
	SendAsFormUrlencoded bool           `json:"send_as_form_urlencoded"`
	Params               []any          `json:"params"`
	ReturnField          map[string]any `json:"return_field,omitempty"`
	MockData             map[string]any `json:"mock_data,omitempty"`
	MockDataEnabled      bool           `json:"mock_data_enabled"`
	Auth                 map[string]any `json:"auth,omitempty"`
	FilterMode           *string        `json:"filter_mode,omitempty"`
	Filters              []any          `json:"filters"`
	FolderID             uuid.UUID      `json:"folder_id"`
	ProjectID            *uuid.UUID     `json:"project_id,omitempty"`
	Active               bool           `json:"active"`
	Inherited            bool           `json:"inherited"`
	DeletedAt            *time.Time     `json:"deleted_at,omitempty"`
	Author               *string        `json:"author,omitempty"`
	Meta                 map[string]any `json:"meta"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

func (Query) FolderEntityType() FolderEntityType {
	return FolderEntityTypeQueries
}

type View struct {
	ID          uuid.UUID      `json:"id"`
	Identity    string         `json:"identity"`
	DisplayName string         `json:"display_name"`
	Description *string        `json:"description,omitempty"`
	IsSystem    bool           `json:"is_system"`
	FolderID    *uuid.UUID     `json:"folder_id,omitempty"`
	ProjectID   *uuid.UUID     `json:"project_id,omitempty"`
	ComponentID *uuid.UUID     `json:"component_id,omitempty"`
	FilterID    *uuid.UUID     `json:"filter_id,omitempty"`
	QueryID     *uuid.UUID     `json:"query_id,omitempty"`
	Inherited   bool           `json:"inherited"`
	Meta        map[string]any `json:"meta"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (View) FolderEntityType() FolderEntityType {
	return FolderEntityTypeDataViews
}
