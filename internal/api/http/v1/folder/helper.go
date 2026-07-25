package folder

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/folders"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) folderIdentityInput(c *fiber.Ctx) folders.FolderIdentityInput {
	return folders.FolderIdentityInput{
		ProjectIdentity: c.Params("project_identity"),
		EntityType:      entities.FolderEntityType(c.Query("entity_type")),
		Identity:        c.Params("folder_identity"),
	}
}

func (h *Handler) newFolderResponse(
	c *fiber.Ctx,
	projectIdentity string,
	folder *entities.RFolder,
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
	folder *entities.RFolder,
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
