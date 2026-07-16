package components_legacy

import "github.com/endge-lab/service-backend/internal/domain/entities"

type ComponentLegacyWithFolder struct {
	ComponentLegacy *entities.RComponentLegacy
	FolderIdentity  string
}
