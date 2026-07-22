package component_legacy

import (
	"context"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/components_legacy"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *components_legacy.ComponentLegacyWithFolder, projectIdentity string) *ComponentLegacyResponse {
	if value == nil {
		return nil
	}

	return newComponentLegacyResponse(value.ComponentLegacy, projectIdentity, value.FolderIdentity)
}

func newComponentLegacyResponse(value *entities.RComponentLegacy, projectIdentity, folderIdentity string) *ComponentLegacyResponse {
	if value == nil {
		return nil
	}

	return &ComponentLegacyResponse{
		ID:              value.ID,
		ProjectIdentity: projectIdentity,
		FolderIdentity:  folderIdentity,
		Identity:        value.Identity,
		DisplayName:     value.DisplayName,
		Description:     value.Description,
		ComponentType:   value.ComponentType,
		Source:          value.Source,
		SourceFormat:    value.SourceFormat,
		PropsSchema:     value.PropsSchema,
		Bindings:        value.Bindings,
		Meta:            value.Meta,
		Active:          value.Active,
		DeletedAt:       value.DeletedAt,
		CreatedAt:       value.CreatedAt,
		UpdatedAt:       value.UpdatedAt,
	}
}

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, components_legacy.ComponentLegacyIdentityInput) error) error {
	if err := fn(c.UserContext(), components_legacy.ComponentLegacyIdentityInput{
		ProjectIdentity:         c.Params("project_identity"),
		ComponentLegacyIdentity: c.Params("component_identity"),
	}); err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
