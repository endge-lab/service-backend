package data_view

import (
	"context"

	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *adapters.DataViewWithRelations, projectIdentity string) *DataViewResponse {
	if value == nil {
		return nil
	}
	return newDataViewResponse(value.DataView, projectIdentity, value.FolderIdentity, value.QueryIdentity)
}

func newDataViewResponse(value *entities.DataView, projectIdentity, folderIdentity, queryIdentity string) *DataViewResponse {
	if value == nil {
		return nil
	}
	return &DataViewResponse{ID: value.ID, ProjectIdentity: projectIdentity, FolderIdentity: folderIdentity, QueryIdentity: queryIdentity, Identity: value.Identity, DisplayName: value.DisplayName, Description: value.Description, ViewType: value.ViewType, Source: value.Source, InputSchema: value.InputSchema, OutputSchema: value.OutputSchema, Meta: value.Meta, Active: value.Active, DeletedAt: value.DeletedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, adapters.DataViewIdentityInput) error) error {
	if err := fn(c.UserContext(), adapters.DataViewIdentityInput{ProjectIdentity: c.Params("project_identity"), DataViewIdentity: c.Params("data_view_identity")}); err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}
