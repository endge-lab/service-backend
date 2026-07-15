package component

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"time"
)

type CreateComponentRequest struct {
	FolderIdentity string                 `json:"folderIdentity" validate:"required"`
	Identity       string                 `json:"identity" validate:"required"`
	DisplayName    string                 `json:"displayName" validate:"required"`
	Description    *string                `json:"description"`
	ComponentType  entities.ComponentType `json:"componentType" validate:"required"`
	Source         string                 `json:"source" validate:"required"`
	PropsSchema    map[string]any         `json:"propsSchema"`
	Bindings       map[string]any         `json:"bindings"`
	Meta           map[string]any         `json:"meta"`
	Active         bool                   `json:"active"`
}
type UpdateComponentRequest struct {
	FolderIdentity string                 `json:"folderIdentity" validate:"required"`
	DisplayName    string                 `json:"displayName" validate:"required"`
	Description    *string                `json:"description"`
	ComponentType  entities.ComponentType `json:"componentType" validate:"required"`
	Source         string                 `json:"source" validate:"required"`
	PropsSchema    map[string]any         `json:"propsSchema"`
	Bindings       map[string]any         `json:"bindings"`
	Meta           map[string]any         `json:"meta"`
	Active         bool                   `json:"active"`
}
type ComponentResponse struct {
	ID              uuid.UUID                      `json:"id"`
	ProjectIdentity string                         `json:"projectIdentity"`
	FolderIdentity  string                         `json:"folderIdentity"`
	Identity        string                         `json:"identity"`
	DisplayName     string                         `json:"displayName"`
	Description     *string                        `json:"description,omitempty"`
	ComponentType   entities.ComponentType         `json:"componentType"`
	Source          string                         `json:"source"`
	SourceFormat    entities.ComponentSourceFormat `json:"sourceFormat"`
	PropsSchema     map[string]any                 `json:"propsSchema"`
	Bindings        map[string]any                 `json:"bindings"`
	Meta            map[string]any                 `json:"meta"`
	Active          bool                           `json:"active"`
	DeletedAt       *time.Time                     `json:"deletedAt"`
	CreatedAt       time.Time                      `json:"createdAt"`
	UpdatedAt       time.Time                      `json:"updatedAt"`
}
type ComponentsListResponse struct {
	Items []*ComponentResponse `json:"items"`
}
