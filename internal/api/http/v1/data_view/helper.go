package data_view

import (
	"context"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *data_views.DataViewWithRelations, projectIdentity string) *DataViewResponse {
	if value == nil {
		return nil
	}
	return newDataViewResponse(value.DataView, projectIdentity, value.FolderIdentity, value.QueryIdentity)
}

func newDataViewResponse(value *entities.RDataView, projectIdentity, folderIdentity, queryIdentity string) *DataViewResponse {
	if value == nil {
		return nil
	}
	return &DataViewResponse{ID: value.ID, ProjectIdentity: projectIdentity, FolderIdentity: folderIdentity, QueryIdentity: queryIdentity, Identity: value.Identity, DisplayName: value.DisplayName, Description: value.Description, ViewType: value.ViewType, Source: value.Source, InputSchema: value.InputSchema, OutputSchema: value.OutputSchema, Meta: value.Meta, Active: value.Active, DeletedAt: value.DeletedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, data_views.DataViewIdentityInput) error) error {
	if err := fn(c.UserContext(), data_views.DataViewIdentityInput{ProjectIdentity: c.Params("project_identity"), DataViewIdentity: c.Params("data_view_identity")}); err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
