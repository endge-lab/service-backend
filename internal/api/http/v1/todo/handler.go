package http

import (
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
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
	createTodoUseCase  usecase.CreateTodoUseCase
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
		createTodoUseCase:  service.CreateTodo,
		validator:          validator,
		logger:             logger.With(zap.String("component", "http_handler")),
		tracer:             tracer,
	}
}

// CreateTodo godoc
// @Summary Создать задачу Todo
// @Description Создает новую задачу Todo и сохраняет ее в PostgreSQL в рамках transaction boundary use case слоя.
// @Tags todo
// @Accept json
// @Produce json
// @Param request body CreateTodoRequest true "Параметры создаваемой задачи"
// @Success 201 {object} TodoResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/todos [post]
func (h *Handler) CreateTodo(c *fiber.Ctx) (err error) {
	logger := logging.WithContext(c.UserContext(), h.logger).With(zap.String("handler", "create_todo"))
	logger.Debug("create todo handler started")

	var request CreateTodoRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}

	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	output, err := h.createTodoUseCase.Execute(c.UserContext(), usecase.CreateTodoInput{
		Title: request.Title,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	logger.Debug("create todo handler completed", zap.String("todo_id", output.Todo.ID))
	return c.Status(fiber.StatusCreated).JSON(NewTodoResponse(output.Todo))
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
