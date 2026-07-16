package components_legacy

import "github.com/endge-lab/service-backend/internal/domain/entities"

type CreateComponentLegacyInput struct {
	ProjectIdentity string
	FolderIdentity  string

	Identity      string
	DisplayName   string
	Description   *string
	ComponentType entities.RComponentLegacyType
	Source        string

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any
	Active      bool
}

type UpdateComponentLegacyInput struct {
	ProjectIdentity         string
	ComponentLegacyIdentity string
	FolderIdentity          string

	DisplayName   string
	Description   *string
	ComponentType entities.RComponentLegacyType
	Source        string

	PropsSchema map[string]any
	Bindings    map[string]any
	Meta        map[string]any
	Active      bool
}

type GetComponentLegacyInput struct {
	ProjectIdentity         string
	ComponentLegacyIdentity string
}

type ComponentLegacyIdentityInput struct {
	ProjectIdentity         string
	ComponentLegacyIdentity string
}

type ListComponentsLegacyInput struct {
	ProjectIdentity string
	FolderIdentity  *string
	ComponentType   *entities.RComponentLegacyType
}
