package ai_assistant

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Capabilities возвращает доступность Workbench и эффективные возможности пользователя.
// @Summary Получить AI capabilities
// @Description Возвращает доступность Workbench, права пользователя, adapters, модели и причину запрета отправки.
// @ID getAICapabilities
// @Tags AI assistant
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Success 200 {object} CapabilitiesResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/capabilities [get]
func (h *Handler) Capabilities(c *fiber.Ctx) error {
	value, err := h.usecase.Capabilities(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// ListConversations возвращает диалоги пользователя в текущем Workspace.
// @Summary Получить AI conversations
// @Description Возвращает cursor-страницу диалогов текущего пользователя в выбранном Workspace.
// @ID listAIConversations
// @Tags AI assistant
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param includeArchived query bool false "Включить архивные диалоги"
// @Param limit query int false "Размер страницы" default(50) maximum(100)
// @Param cursor query string false "Opaque cursor"
// @Success 200 {object} ConversationListResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations [get]
func (h *Handler) ListConversations(c *fiber.Ctx) error {
	limit, err := queryLimit(c, 50, 100)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	page, err := h.usecase.ListConversations(c.UserContext(), c.QueryBool("includeArchived", false), limit, c.Query("cursor"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(ConversationListResponse{Items: page.Items, NextCursor: page.NextCursor})
}

// CreateConversation создаёт активный диалог с выбранной моделью.
// @Summary Создать AI conversation
// @Description Создаёт единственный активный диалог пользователя в Workspace с выбранной enabled-моделью.
// @ID createAIConversation
// @Tags AI assistant
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param request body CreateConversationRequest true "Conversation"
// @Success 201 {object} ConversationResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations [post]
func (h *Handler) CreateConversation(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateConversationRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.CreateConversation(c.UserContext(), request.ModelProfileID)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

// ResetConversation архивирует текущий диалог и атомарно создаёт новый.
// @Summary Сбросить AI conversation
// @Description Атомарно архивирует текущий активный диалог и создаёт новый.
// @ID resetAIConversation
// @Tags AI assistant
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param request body ResetConversationRequest true "Reset"
// @Success 201 {object} ConversationResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations/reset [post]
func (h *Handler) ResetConversation(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[ResetConversationRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.ResetConversation(c.UserContext(), request.CurrentConversationID, request.ModelProfileID)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

// PatchConversation меняет модель только у пустого диалога.
// @Summary Изменить модель AI conversation
// @Description Меняет snapshot модели только у диалога, в котором ещё нет сообщений.
// @ID patchAIConversation
// @Tags AI assistant
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Conversation ID"
// @Param request body PatchConversationRequest true "Новая модель"
// @Success 200 {object} ConversationResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations/{id} [patch]
func (h *Handler) PatchConversation(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[PatchConversationRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.UpdateConversationModel(c.UserContext(), c.Params("id"), request.ModelProfileID)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// ListMessages возвращает страницу сообщений по opaque cursor.
// @Summary Получить сообщения AI conversation
// @Description Возвращает до 50 сообщений по opaque cursor с проверкой ownership.
// @ID listAIConversationMessages
// @Tags AI assistant
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Conversation ID"
// @Param limit query int false "Размер страницы" default(50) maximum(50)
// @Param cursor query string false "Opaque cursor"
// @Success 200 {object} MessageListResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations/{id}/messages [get]
func (h *Handler) ListMessages(c *fiber.Ctx) error {
	limit, err := queryLimit(c, 50, 50)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	page, err := h.usecase.ListMessages(c.UserContext(), c.Params("id"), limit, c.Query("cursor"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(MessageListResponse{Items: page.Items, NextCursor: page.NextCursor})
}

// Run формирует ExportLive snapshot и проксирует server-stream Workbench как SSE.
// @Summary Запустить AI run
// @Description Формирует актуальный ExportLive snapshot и проксирует gRPC stream Workbench как SSE.
// @ID runAIConversation
// @Tags AI assistant
// @Accept json
// @Produce text/event-stream
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param id path string true "Conversation ID"
// @Param request body RunRequest true "Prompt и request ID"
// @Success 200 {string} string "SSE events: started, content_delta, completed или failed"
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 502 {object} shared.ErrorResponse
// @Failure 503 {object} shared.ErrorResponse
// @Failure 504 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/ai/conversations/{id}/runs [post]
func (h *Handler) Run(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[RunRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	conversationID := c.Params("id")
	prepared, err := h.usecase.PrepareRun(c.UserContext(), request.RequestID, conversationID, request.ModelProfileID, request.Prompt)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	streamContext := context.WithoutCancel(c.UserContext())
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(writer *bufio.Writer) {
		writeEvent := func(event entities.AIRunEvent) error {
			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event.Type, payload); err != nil {
				return err
			}
			return writer.Flush()
		}
		if err := h.usecase.RunPrepared(streamContext, prepared, writeEvent); err != nil {
			_ = writeEvent(entities.AIRunEvent{Type: "failed", ErrorCode: domainerrors.CodeOf(err), ErrorMessage: domainerrors.SafeMessageOf(err)})
		}
	}))
	return nil
}

func queryLimit(c *fiber.Ctx, fallback, maximum int) (int, error) {
	raw := c.Query("limit")
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maximum {
		return 0, domainerrors.InvalidInput("pagination.limit_invalid", "limit is invalid")
	}
	return value, nil
}
