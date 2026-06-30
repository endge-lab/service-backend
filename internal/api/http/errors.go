package http

import (
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"

	"github.com/gofiber/fiber/v2"
)

var (
	errRouteNotFound   = domainerrors.NotFound("http.route_not_found", "Маршрут не найден")
	errInvalidBody     = domainerrors.InvalidInput("http.invalid_body", "Некорректное тело запроса")
	errValidationError = domainerrors.InvalidInput("http.validation_failed", "Некорректные поля запроса")
	errInvalidToken    = domainerrors.Unauthorized("auth.invalid_access_token", "Access token недействителен или просрочен")
	errMissingToken    = domainerrors.Unauthorized("auth.access_token_required", "Требуется access token")
	errMissingIdentity = domainerrors.Unauthorized("auth.identity_missing", "В токене отсутствует идентификатор пользователя")
)

func writeErrorResponse(c *fiber.Ctx, err error) error {
	return c.Status(domainerrors.HTTPStatusOf(err)).JSON(ErrorResponse{
		Code:    domainerrors.CodeOf(err),
		Message: domainerrors.SafeMessageOf(err),
		Details: domainerrors.DetailsOf(err),
	})
}
