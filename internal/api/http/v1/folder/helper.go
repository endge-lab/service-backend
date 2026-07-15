package folder

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) folderIdentityInput(c *fiber.Ctx) adapters.FolderIdentityInput {
	return adapters.FolderIdentityInput{
		ProjectIdentity: c.Params("project_identity"),
		EntityType:      entities.FolderEntityType(c.Query("entity_type")),
		Identity:        c.Params("folder_identity"),
	}
}

func (h *Handler) newFolderResponse(
	c *fiber.Ctx,
	projectIdentity string,
	folder *entities.Folder,
) (*FolderResponse, error) {
	if folder == nil || folder.ParentID == nil {
		return NewFolderResponse(folder, projectIdentity, nil), nil
	}

	parent, err := h.folderService.GetByID(c.UserContext(), *folder.ParentID)
	if err != nil {
		return nil, err
	}

	parentIdentity := parent.Identity
	return NewFolderResponse(folder, projectIdentity, &parentIdentity), nil
}

func NewFolderResponse(
	folder *entities.Folder,
	projectIdentity string,
	parentIdentity *string,
) *FolderResponse {
	if folder == nil {
		return nil
	}

	return &FolderResponse{
		ID:              folder.ID,
		ProjectIdentity: projectIdentity,
		EntityType:      folder.EntityType,
		Identity:        folder.Identity,
		DisplayName:     folder.DisplayName,
		Description:     folder.Description,
		ParentIdentity:  parentIdentity,
		IsRoot:          folder.IsRoot,
		IsSystem:        folder.IsSystem,
		DeletedAt:       folder.DeletedAt,
		Meta:            folder.Meta,
		CreatedAt:       folder.CreatedAt,
		UpdatedAt:       folder.UpdatedAt,
	}
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}
