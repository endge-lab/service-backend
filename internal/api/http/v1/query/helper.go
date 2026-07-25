package query

import (
	"context"

	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/queries"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) response(value *queries.QueryWithFolder, projectIdentity string) *QueryResponse {
	if value == nil {
		return nil
	}
	return newQueryResponse(value.Query, projectIdentity, value.FolderIdentity)
}

func newQueryResponse(value *entities.RQuery, projectIdentity, folderIdentity string) *QueryResponse {
	if value == nil {
		return nil
	}
	return &QueryResponse{ID: value.ID, ProjectIdentity: projectIdentity, FolderIdentity: folderIdentity, Identity: value.Identity, DisplayName: value.DisplayName, Description: value.Description, QueryType: value.QueryType, Source: value.Source, Params: value.Params, Headers: value.Headers, Auth: value.Auth, TimeoutMS: value.TimeoutMS, MockData: value.MockData, MockDataEnabled: value.MockDataEnabled, Meta: value.Meta, Active: value.Active, DeletedAt: value.DeletedAt, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, queries.QueryIdentityInput) error) error {
	if err := fn(c.UserContext(), queries.QueryIdentityInput{ProjectIdentity: c.Params("project_identity"), QueryIdentity: c.Params("query_identity")}); err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
