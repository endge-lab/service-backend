package converter

import (
	"context"

	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *adapters.ConverterWithFolder, projectIdentity string) *ConverterResponse {
	if value == nil {
		return nil
	}

	return newConverterResponse(value.Converter, projectIdentity, value.FolderIdentity)
}

func newConverterResponse(value *entities.Converter, projectIdentity, folderIdentity string) *ConverterResponse {
	if value == nil {
		return nil
	}

	return &ConverterResponse{
		ID:              value.ID,
		ProjectIdentity: projectIdentity,
		FolderIdentity:  folderIdentity,
		Identity:        value.Identity,
		DisplayName:     value.DisplayName,
		Description:     value.Description,
		ConverterType:   value.ConverterType,
		Source:          value.Source,
		IsSystem:        value.IsSystem,
		Meta:            value.Meta,
		Active:          value.Active,
		DeletedAt:       value.DeletedAt,
		CreatedAt:       value.CreatedAt,
		UpdatedAt:       value.UpdatedAt,
	}
}

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, adapters.ConverterIdentityInput) error) error {
	if err := fn(c.UserContext(), adapters.ConverterIdentityInput{
		ProjectIdentity:   c.Params("project_identity"),
		ConverterIdentity: c.Params("converter_identity"),
	}); err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}
