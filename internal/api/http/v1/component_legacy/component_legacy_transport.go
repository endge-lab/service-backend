package component_legacy

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"time"
)

type CreateComponentLegacyRequest struct {
	FolderIdentity string                        `json:"folderIdentity" validate:"required"`
	Identity       string                        `json:"identity" validate:"required"`
	DisplayName    string                        `json:"displayName" validate:"required"`
	Description    *string                       `json:"description"`
	ComponentType  entities.RComponentLegacyType `json:"componentType" validate:"required"`
	Source         string                        `json:"source" validate:"required"`
	PropsSchema    map[string]any                `json:"propsSchema"`
	Bindings       map[string]any                `json:"bindings"`
	Meta           map[string]any                `json:"meta"`
	Active         bool                          `json:"active"`
}
type UpdateComponentLegacyRequest struct {
	FolderIdentity string                        `json:"folderIdentity" validate:"required"`
	DisplayName    string                        `json:"displayName" validate:"required"`
	Description    *string                       `json:"description"`
	ComponentType  entities.RComponentLegacyType `json:"componentType" validate:"required"`
	Source         string                        `json:"source" validate:"required"`
	PropsSchema    map[string]any                `json:"propsSchema"`
	Bindings       map[string]any                `json:"bindings"`
	Meta           map[string]any                `json:"meta"`
	Active         bool                          `json:"active"`
}
type ComponentLegacyResponse struct {
	ID              uuid.UUID                             `json:"id"`
	ProjectIdentity string                                `json:"projectIdentity"`
	FolderIdentity  string                                `json:"folderIdentity"`
	Identity        string                                `json:"identity"`
	DisplayName     string                                `json:"displayName"`
	Description     *string                               `json:"description,omitempty"`
	ComponentType   entities.RComponentLegacyType         `json:"componentType"`
	Source          string                                `json:"source"`
	SourceFormat    entities.RComponentLegacySourceFormat `json:"sourceFormat"`
	PropsSchema     map[string]any                        `json:"propsSchema"`
	Bindings        map[string]any                        `json:"bindings"`
	Meta            map[string]any                        `json:"meta"`
	Active          bool                                  `json:"active"`
	DeletedAt       *time.Time                            `json:"deletedAt"`
	CreatedAt       time.Time                             `json:"createdAt"`
	UpdatedAt       time.Time                             `json:"updatedAt"`
}
type ComponentsLegacyListResponse struct {
	Items []*ComponentLegacyResponse `json:"items"`
}
