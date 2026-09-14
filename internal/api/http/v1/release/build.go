package release

import (
	"encoding/json"
	"io"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/releases"
	"github.com/gofiber/fiber/v2"
)

// CreateFromBuild creates a release from a saved build and optionally creates its commit.
// @Summary Создать релиз из сборки
// @ID createReleaseFromBuild
// @Tags Релизы
// @Accept multipart/form-data
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param metadata formData string true "JSON: identity, displayName, description, sourceCommitId, workspaceId, generation, headSequence, commitMessage, buildMetadata"
// @Param bundle formData file true "Gzip Endge Bundle, максимум 15 MiB"
// @Success 201 {object} Response
// @Failure 400 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/releases/from-build [post]
func (h *Handler) CreateFromBuild(c *fiber.Ctx) error {
	var input resourceusecase.CreateFromBuildInput
	metadata := c.FormValue("metadata")
	if len(metadata) > 64*1024 || json.Unmarshal([]byte(metadata), &input) != nil {
		return respond.WriteErrorResponse(c, domainerrors.InvalidInput("build_metadata_invalid", "Invalid build metadata"))
	}
	if len(input.DisplayName) > 255 || (input.Description != nil && len(*input.Description) > 16000) || len(input.CommitMessage) > 4000 {
		return respond.WriteErrorResponse(c, domainerrors.InvalidInput("release_text_too_long", "Release text exceeds limit"))
	}
	file, err := c.FormFile("bundle")
	if err != nil || file.Size > resourceusecase.MaxReleaseBuildBytes {
		return respond.WriteErrorResponse(c, domainerrors.InvalidInput("release_build_required", "Attach one Gzip Bundle, at most 15 MiB"))
	}
	reader, err := file.Open()
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	defer func() { _ = reader.Close() }()
	input.Data, err = io.ReadAll(io.LimitReader(reader, resourceusecase.MaxReleaseBuildBytes+1))
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.CreateFromBuild(c.UserContext(), input)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusCreated, *value)
}

// ExportBuild downloads the optional compiled release Bundle.
// @Summary Скачать Bundle релиза
// @ID exportReleaseBuild
// @Tags Релизы
// @Produce application/gzip
// @Param X-Endge-Workspace header string true "Identity рабочего пространства"
// @Param identity path string true "Identity релиза"
// @Success 200 {file} binary
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/releases/{identity}/bundle [get]
func (h *Handler) ExportBuild(c *fiber.Ctx) error {
	identity, err := shared.PathParam(c, "identity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	data, release, err := h.usecase.GetBuild(c.UserContext(), identity)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderContentType, "application/gzip")
	c.Set(fiber.HeaderCacheControl, "private, no-cache")
	c.Set(fiber.HeaderVary, "X-Endge-Workspace, Authorization, Cookie")
	if release.BuildMetadata != nil {
		c.Set(fiber.HeaderETag, `"`+release.BuildMetadata.Checksum+`"`)
	}
	c.Attachment(shared.SafeAttachmentName(release.Identity, "release") + ".endge-bundle.gz")
	return c.Send(data)
}
