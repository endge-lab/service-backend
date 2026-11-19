package modules

import (
	"io"
	"strconv"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/gofiber/fiber/v2"
)

type ListResponse struct {
	Items []entities.Module `json:"items"`
	Total int               `json:"total"`
}

type VersionResponse struct {
	entities.ModuleVersion
	ArchiveURL string    `json:"archiveUrl"`
	Assets     AssetURLs `json:"assets"`
}

type AssetURLs struct {
	JS  string `json:"js"`
	CSS string `json:"css,omitempty"`
}

type PatchRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func setRevision(c *fiber.Ctx, value int64) {
	c.Set(fiber.HeaderETag, strconv.Quote(strconv.FormatInt(value, 10)))
}

func (h *Handler) readUpload(c *fiber.Ctx) ([]byte, error) {
	if err := h.usecase.AuthorizePublish(c.UserContext()); err != nil {
		return nil, err
	}
	header, err := c.FormFile("archive")
	if err != nil || header.Size == 0 || header.Size > maxArchiveSize {
		return nil, domainerrors.InvalidInput("module_archive_invalid", "A ZIP archive of at most 15 MiB is required")
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxArchiveSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxArchiveSize {
		return nil, domainerrors.InvalidInput("module_archive_invalid", "ZIP archive exceeds 15 MiB")
	}
	return data, nil
}
