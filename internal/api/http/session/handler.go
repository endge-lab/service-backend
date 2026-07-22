package session

import (
	"strings"

	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/observability"
	usecasesession "github.com/endge-lab/service-backend/internal/usecase/session"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ErrorResponse = respond.ErrorResponse
type Handler struct {
	loadSessionUseCase UseCase
	validator          appvalidator.Validator
	observer           observability.Observer
}

func NewHandler(
	useCase UseCase,
	validator appvalidator.Validator,
	core *observability.Core,
	metrics *httpobservability.HandlerMetrics,
) *Handler {
	observer := core.For(observability.LayerHandler, "session_http_handler").WithRecorder(metrics)
	return &Handler{
		loadSessionUseCase: useCase,
		validator:          validator,
		observer:           observer,
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
	logger := logging.WithContext(c.UserContext(), h.observer.Logger()).With(zap.String("handler", "load_session"))
	logger.Debug("load session handler started")

	identity, ok := httpmiddleware.IdentityFromContext(c.UserContext())
	if !ok || strings.TrimSpace(identity.AuthUserID) == "" {
		return respond.WriteErrorResponse(c, respond.ErrMissingIdentity)
	}

	response, err := h.loadSessionUseCase.Execute(c.UserContext(), usecasesession.LoadSessionInput{
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
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}

	logger.Debug("load session handler completed", zap.String("service_user_id", response.User.ID))
	return c.JSON(NewSessionResponse(response))
}
