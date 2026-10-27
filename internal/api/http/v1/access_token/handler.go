package access_token

import (
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/access_tokens"
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

// Create выпускает персональный токен агента для текущего пользователя.
// @Summary Выпустить персональный токен доступа
// @Description Выпускает токен `endge_pat_…` от имени текущего пользователя для одного workspace. Секрет возвращается только в этом ответе; в БД хранится SHA-256. Срок обязателен и ограничен ACCESS_TOKEN_MAX_TTL (по умолчанию 30 дней). Scope по умолчанию `read`; флаги `commit` и `delete` требуют `write`, а `write` — роли Editor или выше. Токеном нельзя выпускать, просматривать и отзывать токены: нужна cookie- или JWT-сессия.
// @ID createAccessToken
// @Tags Персональные токены
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Параметры токена"
// @Success 201 {object} CreateResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-tokens [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	created, err := h.usecase.Create(c.UserContext(), resourceusecase.CreateInput{
		WorkspaceIdentity: request.WorkspaceIdentity, Name: request.Name, Description: request.Description,
		Scope: request.Scope, CanCommit: request.Commit, CanDelete: request.Delete, ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Status(fiber.StatusCreated).JSON(CreateResponse{Token: created.Secret, AccessToken: newTokenResponse(created.Token, time.Now())})
}

// List возвращает персональные токены текущего пользователя.
// @Summary Получить персональные токены
// @Description Возвращает токены текущего пользователя без секретов, включая истёкшие и отозванные. Platform Admin с `owner=all` получает токены всех пользователей.
// @ID listAccessTokens
// @Tags Персональные токены
// @Produce json
// @Param owner query string false "me (по умолчанию) или all для Platform Admin" Enums(me, all)
// @Success 200 {object} ListResponse
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-tokens [get]
func (h *Handler) List(c *fiber.Ctx) error {
	owner := c.Query("owner", "me")
	if owner != "me" && owner != "all" {
		return respond.WriteErrorResponse(c, domainerrors.WithDetails(domainerrors.InvalidInput("query_owner_invalid", "owner must be me or all"), map[string]any{"field": "owner"}))
	}
	items, err := h.usecase.List(c.UserContext(), owner == "all")
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	now := time.Now()
	response := ListResponse{Items: make([]TokenResponse, 0, len(items)), MaxTTLSeconds: int64(h.usecase.MaxTTL() / time.Second)}
	for _, item := range items {
		response.Items = append(response.Items, newTokenResponse(item, now))
	}
	return c.JSON(response)
}

// Revoke отзывает персональный токен.
// @Summary Отозвать персональный токен
// @Description Отзывает свой токен; Platform Admin может отозвать любой. Повторный отзыв успешен и сохраняет первое время отзыва. Отозванный токен получает 401 `access_token_revoked`.
// @ID revokeAccessToken
// @Tags Персональные токены
// @Param id path string true "ID токена" format(uuid)
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/access-tokens/{id} [delete]
func (h *Handler) Revoke(c *fiber.Ctx) error {
	if err := h.usecase.Revoke(c.UserContext(), c.Params("id")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
