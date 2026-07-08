package http

import (
	"strings"

	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/middleware"
	"github.com/endge-lab/service-backend/internal/usecase"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ErrorResponse = transport.ErrorResponse
type Handler struct {
	loadSessionUseCase usecase.LoadSessionUseCase
	validator          appvalidator.Validator
	logger             *zap.Logger
	tracer             trace.Tracer
}

func NewHandler(
	service *usecase.Service,
	validator appvalidator.Validator,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Handler {
	return &Handler{
		loadSessionUseCase: service.LoadSession,
		validator:          validator,
		logger:             logger.With(zap.String("component", "http_handler")),
		tracer:             tracer,
	}
}

// LoadSession godoc
// @Summary Получить текущую сессию
// @Description Возвращает информацию о JWT-сессии и локальной user-проекции сервиса.
// @Tags session
// @Accept json
// @Produce json
// @Success 200 {object} SessionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/session/me [get]
func (h *Handler) LoadSession(c *fiber.Ctx) (err error) {
	logger := logging.WithContext(c.UserContext(), h.logger).With(zap.String("handler", "load_session"))
	logger.Debug("load session handler started")

	identity, ok := middleware.IdentityFromContext(c.UserContext())
	if !ok || strings.TrimSpace(identity.AuthUserID) == "" {
		return transport.WriteErrorResponse(c, transport.ErrMissingIdentity)
	}

	response, err := h.loadSessionUseCase.Execute(c.UserContext(), usecase.LoadSessionInput{
		AuthUserID:  identity.AuthUserID,
		Username:    identity.Username,
		DisplayName: identity.DisplayName,
		Role:        identity.Role,
		SessionID:   identity.SessionID,
		App:         identity.App,
		Platform:    identity.Platform,
		Scope:       identity.Scope,
		ExpiresAt:   identity.ExpiresAt,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	logger.Debug("load session handler completed", zap.String("service_user_id", response.User.ID))
	return c.JSON(NewSessionResponse(response))
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}

func (h *Handler) respondDomainError(c *fiber.Ctx, err error) error {
	return h.respondUnexpectedError(c, err)
}

func (h *Handler) respondUnexpectedError(c *fiber.Ctx, err error) error {
	fields := []zap.Field{
		zap.Error(err),
		zap.String("error_code", domainerrors.CodeOf(err)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}

	logger := logging.WithContext(c.UserContext(), h.logger)

	if domainerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
		logger.Error("unexpected request transport", fields...)
	} else {
		logger.Warn("request completed with business transport", fields...)
	}

	return transport.WriteErrorResponse(c, err)
}
