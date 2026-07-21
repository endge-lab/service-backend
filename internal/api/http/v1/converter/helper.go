package converter

import (
	"context"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *converters.ConverterWithFolder, projectIdentity string) *ConverterResponse {
	if value == nil {
		return nil
	}

	return newConverterResponse(value.Converter, projectIdentity, value.FolderIdentity)
}

func newConverterResponse(value *entities.RConverter, projectIdentity, folderIdentity string) *ConverterResponse {
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

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, converters.ConverterIdentityInput) error) error {
	if err := fn(c.UserContext(), converters.ConverterIdentityInput{
		ProjectIdentity:   c.Params("project_identity"),
		ConverterIdentity: c.Params("converter_identity"),
	}); err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
