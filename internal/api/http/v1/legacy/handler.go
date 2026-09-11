package legacy

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает временные миграционные API.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт обработчик Legacy API.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// RebuildWorkspaceFoldersFromFrontend пересоздаёт Workspace-папки по Frontend-проекции.
// @Summary Пересоздать Workspace-папки из Frontend
// @Description Удаляет текущие пользовательские Workspace-папки, копирует Frontend-корни с подпапками и перепривязывает документы. Не изменяет документы и их folderId.
// @ID legacyRebuildWorkspaceFoldersFromFrontend
// @Tags Legacy
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body RebuildWorkspaceFoldersRequest true "Явное подтверждение целевого Workspace"
// @Success 200 {object} RebuildWorkspaceFoldersResponse "Результат пересоздания"
// @Failure 400 {object} shared.ErrorResponse "Некорректное подтверждение"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуется роль Workspace Admin"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/legacy/workspace-folders/rebuild-from-frontend [post]
func (h *Handler) RebuildWorkspaceFoldersFromFrontend(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[RebuildWorkspaceFoldersRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	result, err := h.usecase.RebuildWorkspaceFoldersFromFrontend(c.UserContext(), request.Confirmation)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(RebuildWorkspaceFoldersResponse{
		FoldersDeleted: result.FoldersDeleted, FoldersCreated: result.FoldersCreated,
		DocumentsRelinked: result.DocumentsRelinked,
	})
}
