package component

import (
	"context"

	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *adapters.ComponentWithFolder, projectIdentity string) *ComponentResponse {
	if value == nil {
		return nil
	}

	return newComponentResponse(value.Component, projectIdentity, value.FolderIdentity)
}

func newComponentResponse(value *entities.Component, projectIdentity, folderIdentity string) *ComponentResponse {
	if value == nil {
		return nil
	}

	return &ComponentResponse{
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

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, adapters.ComponentIdentityInput) error) error {
	if err := fn(c.UserContext(), adapters.ComponentIdentityInput{
		ProjectIdentity:   c.Params("project_identity"),
		ComponentIdentity: c.Params("component_identity"),
	}); err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}
