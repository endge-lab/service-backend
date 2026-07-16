package respond

import (
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-kit-go/pkg/logging"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var (
	ErrRouteNotFound   = domainerrors.NotFound("not_found", "Маршрут не найден")
	ErrInvalidBody     = domainerrors.InvalidInput("validation_error", "Некорректное тело запроса")
	ErrValidationError = domainerrors.InvalidInput("validation_error", "Некорректные поля запроса")
	ErrInvalidToken    = domainerrors.Unauthorized("auth.invalid_access_token", "Access token недействителен или просрочен")
	ErrMissingToken    = domainerrors.Unauthorized("auth.access_token_required", "Требуется access token")
	ErrMissingIdentity = domainerrors.Unauthorized("auth.identity_missing", "В токене отсутствует идентификатор пользователя")
)

func WriteErrorResponse(c *fiber.Ctx, err error) error {
	return c.Status(domainerrors.HTTPStatusOf(err)).JSON(ErrorResponse{
		Code:    domainerrors.CodeOf(err),
		Message: domainerrors.SafeMessageOf(err),
		Details: domainerrors.DetailsOf(err),
	})
}

func RespondDomainError(c *fiber.Ctx, logger *zap.Logger, err error) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	fields := []zap.Field{
		zap.Error(err),
		zap.String("error_code", domainerrors.CodeOf(err)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}

	logger = logging.WithContext(c.UserContext(), logger)
	if domainerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
		logger.Error("unexpected request transport", fields...)
	} else {
		logger.Warn("request completed with business transport", fields...)
	}

	return WriteErrorResponse(c, err)
}
