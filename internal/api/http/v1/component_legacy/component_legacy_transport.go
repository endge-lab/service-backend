package component_legacy

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/google/uuid"
	"time"
)

type CreateComponentLegacyRequest struct {
	FolderIdentity string                        `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-components-legacy"`
	Identity       string                        `json:"identity" validate:"required,min=1,max=160" example:"example-user-card"`
	DisplayName    string                        `json:"displayName" validate:"required,min=1,max=255" example:"Example user card"`
	Description    *string                       `json:"description" example:"Displays a user card"`
	ComponentType  entities.RComponentLegacyType `json:"componentType" validate:"required,oneof=component-sfc" enums:"component-sfc" example:"component-sfc"`
	Source         string                        `json:"source" validate:"required,min=1" example:"<template><article>{{ user.name }}</article></template>"`
	PropsSchema    map[string]any                `json:"propsSchema"`
	Bindings       map[string]any                `json:"bindings"`
	Meta           map[string]any                `json:"meta"`
	Active         bool                          `json:"active" example:"true"`
}
type UpdateComponentLegacyRequest struct {
	FolderIdentity string                        `json:"folderIdentity" validate:"required,min=1,max=160" example:"shared-components-legacy"`
	DisplayName    string                        `json:"displayName" validate:"required,min=1,max=255" example:"User card"`
	Description    *string                       `json:"description" example:"Displays a user card"`
	ComponentType  entities.RComponentLegacyType `json:"componentType" validate:"required,oneof=component-sfc" enums:"component-sfc" example:"component-sfc"`
	Source         string                        `json:"source" validate:"required,min=1" example:"<template><article>{{ user.name }}</article></template>"`
	PropsSchema    map[string]any                `json:"propsSchema"`
	Bindings       map[string]any                `json:"bindings"`
	Meta           map[string]any                `json:"meta"`
	Active         bool                          `json:"active" example:"true"`
}
type ComponentLegacyResponse struct {
	ID              uuid.UUID                             `json:"id" example:"00000000-0000-4000-8000-000000000051"`
	ProjectIdentity string                                `json:"projectIdentity" example:"demo-project"`
	FolderIdentity  string                                `json:"folderIdentity" example:"shared-components-legacy"`
	Identity        string                                `json:"identity" example:"user-card"`
	DisplayName     string                                `json:"displayName" example:"User card"`
	Description     *string                               `json:"description,omitempty" example:"Displays a user card"`
	ComponentType   entities.RComponentLegacyType         `json:"componentType" enums:"component-sfc" example:"component-sfc"`
	Source          string                                `json:"source" example:"<template><article>{{ user.name }}</article></template>"`
	SourceFormat    entities.RComponentLegacySourceFormat `json:"sourceFormat" example:"vue"`
	PropsSchema     map[string]any                        `json:"propsSchema"`
	Bindings        map[string]any                        `json:"bindings"`
	Meta            map[string]any                        `json:"meta"`
	Active          bool                                  `json:"active" example:"true"`
	DeletedAt       *time.Time                            `json:"deletedAt" example:"2026-07-23T10:00:00Z"`
	CreatedAt       time.Time                             `json:"createdAt" example:"2026-07-23T10:00:00Z"`
	UpdatedAt       time.Time                             `json:"updatedAt" example:"2026-07-23T10:00:00Z"`
}
type ComponentsLegacyListResponse struct {
	Items []*ComponentLegacyResponse `json:"items"`
}
