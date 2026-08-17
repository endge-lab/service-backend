package document_move

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает атомарные операции над несколькими domain-документами.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт обработчик массового перемещения документов.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Move атомарно переносит несколько документов в одну папку.
// @Summary Переместить документы в одну папку
// @Description Атомарно переносит до 500 документов в одну папку с optimistic locking и общей mutation batch.
// @ID moveDomainDocuments
// @Tags Домен
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body MoveRequest true "Документы и папка назначения"
// @Success 200 {object} MoveResponse "Актуальные документы после перемещения"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Документ или папка не найдены"
// @Failure 409 {object} shared.ErrorResponse "Конфликт revision"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/documents/move [post]
func (h *Handler) Move(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[MoveRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	input := documents.MoveDocumentsInput{
		Documents:      make([]documents.MoveDocumentInput, 0, len(request.Documents)),
		FolderIdentity: request.FolderIdentity,
	}
	for _, item := range request.Documents {
		input.Documents = append(input.Documents, documents.MoveDocumentInput{
			Collection: item.Collection, Identity: item.Identity, ExpectedRevision: item.ExpectedRevision,
		})
	}

	result, err := h.usecase.MoveDocuments(c.UserContext(), input)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := newMoveResponse(result)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(response)
}
