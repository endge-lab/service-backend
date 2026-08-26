package ai_catalog

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/ai_catalog"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Adapters возвращает поддерживаемые backend-ом типы AI-провайдеров.
// @Summary Получить AI adapters
// @Description Возвращает фиксированный список adapter IDs, поддерживаемых платформой.
// @ID listAIProviderAdapters
// @Tags AI catalog
// @Produce json
// @Success 200 {object} AdaptersResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-adapters [get]
func (h *Handler) Adapters(c *fiber.Ctx) error {
	items, err := h.usecase.Adapters(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(AdaptersResponse{Items: items})
}

// ListConnections возвращает каталог connections без credential.
// @Summary Получить AI connections
// @Description Возвращает connections без значения credential, только с признаком его наличия.
// @ID listAIProviderConnections
// @Tags AI catalog
// @Produce json
// @Success 200 {object} ConnectionsResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-connections [get]
func (h *Handler) ListConnections(c *fiber.Ctx) error {
	items, err := h.usecase.ListConnections(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(ConnectionsResponse{Items: items, Total: len(items)})
}

// CreateConnection создаёт connection; credential сохраняется зашифрованным.
// @Summary Создать AI connection
// @Description Создаёт connection платформы и сохраняет переданный credential в зашифрованном виде.
// @ID createAIProviderConnection
// @Tags AI catalog
// @Accept json
// @Produce json
// @Param request body CreateConnectionRequest true "Connection"
// @Success 201 {object} ConnectionResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-connections [post]
func (h *Handler) CreateConnection(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateConnectionRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.CreateConnection(c.UserContext(), request.Name, request.Adapter, request.BaseURL, request.Credential, request.Enabled)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

// PatchConnection изменяет несекретные настройки connection.
// @Summary Изменить AI connection
// @Description Частично изменяет имя, base URL или enabled-состояние connection.
// @ID patchAIProviderConnection
// @Tags AI catalog
// @Accept json
// @Produce json
// @Param id path string true "Connection ID"
// @Param request body PatchConnectionRequest true "Изменяемые поля"
// @Success 200 {object} ConnectionResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-connections/{id} [patch]
func (h *Handler) PatchConnection(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[PatchConnectionRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PatchConnection(c.UserContext(), c.Params("id"), resourceusecase.ConnectionPatch{
		Name: request.Name, BaseURL: request.BaseURL, Enabled: request.Enabled,
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// ReplaceCredential атомарно заменяет зашифрованный credential connection.
// @Summary Заменить credential AI connection
// @Description Заменяет зашифрованный credential, не возвращая его значение в ответе.
// @ID replaceAIProviderConnectionCredential
// @Tags AI catalog
// @Accept json
// @Produce json
// @Param id path string true "Connection ID"
// @Param request body ReplaceCredentialRequest true "Новый credential"
// @Success 200 {object} ConnectionResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-connections/{id}/credential [put]
func (h *Handler) ReplaceCredential(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[ReplaceCredentialRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.ReplaceCredential(c.UserContext(), c.Params("id"), request.Credential)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// DeleteConnection физически удаляет connection, credential и связанные profiles.
// @Summary Удалить AI connection
// @Description Физически удаляет connection, credential и все связанные model profiles.
// @ID deleteAIProviderConnection
// @Tags AI catalog
// @Param id path string true "Connection ID"
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/provider-connections/{id} [delete]
func (h *Handler) DeleteConnection(c *fiber.Ctx) error {
	if err := h.usecase.DeleteConnection(c.UserContext(), c.Params("id")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListModels возвращает model profiles платформы.
// @Summary Получить AI model profiles
// @Description Возвращает все model profiles платформы без soft-delete фильтрации.
// @ID listAIModelProfiles
// @Tags AI catalog
// @Produce json
// @Success 200 {object} ModelsResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/model-profiles [get]
func (h *Handler) ListModels(c *fiber.Ctx) error {
	items, err := h.usecase.ListModels(c.UserContext(), false)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(ModelsResponse{Items: items, Total: len(items)})
}

// CreateModel создаёт model profile без автоматического default.
// @Summary Создать AI model profile
// @Description Создаёт model profile; default назначается только явно.
// @ID createAIModelProfile
// @Tags AI catalog
// @Accept json
// @Produce json
// @Param request body CreateModelRequest true "Model profile"
// @Success 201 {object} ModelResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/model-profiles [post]
func (h *Handler) CreateModel(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateModelRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.CreateModel(c.UserContext(), request.ConnectionID, request.ProviderModelID, request.DisplayName, request.Enabled, request.Default)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

// PatchModel изменяет model profile и обеспечивает единственный enabled default.
// @Summary Изменить AI model profile
// @Description Частично изменяет profile и обеспечивает не более одного enabled default.
// @ID patchAIModelProfile
// @Tags AI catalog
// @Accept json
// @Produce json
// @Param id path string true "Model profile ID"
// @Param request body PatchModelRequest true "Изменяемые поля"
// @Success 200 {object} ModelResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/model-profiles/{id} [patch]
func (h *Handler) PatchModel(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[PatchModelRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PatchModel(c.UserContext(), c.Params("id"), resourceusecase.ModelPatch{
		ProviderModelID: request.ProviderModelID, DisplayName: request.DisplayName, Enabled: request.Enabled, Default: request.Default,
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// DeleteModel физически удаляет model profile без выбора нового default.
// @Summary Удалить AI model profile
// @Description Физически удаляет profile и не выбирает новый default автоматически.
// @ID deleteAIModelProfile
// @Tags AI catalog
// @Param id path string true "Model profile ID"
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/model-profiles/{id} [delete]
func (h *Handler) DeleteModel(c *fiber.Ctx) error {
	if err := h.usecase.DeleteModel(c.UserContext(), c.Params("id")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
