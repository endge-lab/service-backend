package access_control

import (
	"strconv"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_control"
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

// SearchUsers лениво ищет активных пользователей по username.
// @Summary Найти пользователей для назначения доступа
// @Description Возвращает страницу активных пользователей, доступных текущему администратору для назначения ролей.
// @ID searchServiceUsers
// @Tags Управление доступом
// @Produce json
// @Param q query string true "Префикс username" minlength(2)
// @Param workspaceIdentity query string false "Workspace для проверки полномочий"
// @Param cursor query string false "Cursor следующей страницы"
// @Param limit query int false "Размер страницы" minimum(1) maximum(50)
// @Success 200 {object} UserListResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/service-users/search [get]
func (h *Handler) SearchUsers(c *fiber.Ctx) error {
	page, err := h.usecase.SearchUsers(c.UserContext(), c.Query("q"), c.Query("workspaceIdentity"), c.Query("cursor"), queryLimit(c))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items := make([]UserResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, newUserResponse(item))
	}
	return c.JSON(UserListResponse{Items: items, NextCursor: page.NextCursor})
}

// List возвращает прямые назначения выбранного scope.
// @Summary Получить назначения доступа
// @Description Возвращает страницу прямых назначений ролей платформы или выбранного рабочего пространства.
// @ID listAccessGrants
// @Tags Управление доступом
// @Produce json
// @Param scopeType query string true "platform или workspace"
// @Param workspaceIdentity query string false "Workspace identity"
// @Param userId query string false "Фильтр пользователя для Platform Admin" format(uuid)
// @Param q query string false "Префикс username"
// @Param cursor query string false "Cursor следующей страницы"
// @Param limit query int false "Размер страницы" minimum(1) maximum(100)
// @Success 200 {object} GrantListResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-grants [get]
func (h *Handler) List(c *fiber.Ctx) error {
	page, err := h.usecase.List(c.UserContext(), resourceusecase.ListInput{
		ScopeType: c.Query("scopeType"), WorkspaceIdentity: c.Query("workspaceIdentity"),
		UserID: c.Query("userId"), Query: c.Query("q"), Cursor: c.Query("cursor"), Limit: queryLimit(c),
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items := make([]GrantResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, newGrantResponse(item))
	}
	return c.JSON(GrantListResponse{Items: items, NextCursor: page.NextCursor})
}

// Put создаёт или заменяет назначение роли.
// @Summary Назначить роль
// @Description Создаёт прямое назначение роли или заменяет существующую роль пользователя в выбранном scope.
// @ID putAccessGrant
// @Tags Управление доступом
// @Accept json
// @Produce json
// @Param request body PutRequest true "Назначение роли"
// @Success 200 {object} GrantResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-grants [put]
func (h *Handler) Put(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[PutRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	grant, err := h.usecase.Put(c.UserContext(), resourceusecase.PutInput{
		UserID: request.UserID, ScopeType: request.ScopeType,
		WorkspaceIdentity: request.WorkspaceIdentity, Role: request.Role,
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(newGrantResponse(*grant))
}

// Delete отзывает прямое назначение роли.
// @Summary Отозвать роль
// @Description Удаляет прямое назначение роли по идентификатору.
// @ID deleteAccessGrant
// @Tags Управление доступом
// @Param id path string true "Grant ID" format(uuid)
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-grants/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.usecase.Delete(c.UserContext(), c.Params("id")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Bulk массово назначает Workspace-роли.
// @Summary Массово назначить роли Workspace
// @Description Массово создаёт или обновляет назначения Workspace-роли для выбранных рабочих пространств.
// @ID bulkPutWorkspaceAccessGrants
// @Tags Управление доступом
// @Accept json
// @Produce json
// @Param request body BulkRequest true "Массовое назначение"
// @Success 200 {object} BulkResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-grants/bulk-workspaces [post]
func (h *Handler) Bulk(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[BulkRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	result, err := h.usecase.Bulk(c.UserContext(), resourceusecase.BulkInput{
		UserID: request.UserID, Role: request.Role, SelectionType: request.Selection.Type,
		WorkspaceIdentities: request.Selection.WorkspaceIdentities,
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(BulkResponse{Affected: result.Affected, Created: result.Created, Updated: result.Updated})
}

func queryLimit(c *fiber.Ctx) int {
	value, _ := strconv.Atoi(c.Query("limit"))
	return value
}
