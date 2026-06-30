package http

import (
	"os"
	"strings"

	"github.com/endge-lab/service-template-go/internal/config"
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"github.com/endge-lab/service-template-go/internal/middleware"
	"github.com/endge-lab/service-template-go/internal/usecase"
	"github.com/endge-lab/service-template-go/internal/util"
	appvalidator "github.com/endge-lab/service-template-go/internal/validator"

	otelfiber "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler struct {
	loadSessionUseCase usecase.LoadSessionUseCase
	createTodoUseCase  usecase.CreateTodoUseCase
	validator          appvalidator.Validator
	cfg                *config.Config
	logger             *zap.Logger
	tracer             trace.Tracer
}

func NewHandler(
	loadSessionUseCase usecase.LoadSessionUseCase,
	createTodoUseCase usecase.CreateTodoUseCase,
	validator appvalidator.Validator,
	cfg *config.Config,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Handler {
	return &Handler{
		loadSessionUseCase: loadSessionUseCase,
		createTodoUseCase:  createTodoUseCase,
		validator:          validator,
		cfg:                cfg,
		logger:             logger.With(zap.String("component", "http_handler")),
		tracer:             tracer,
	}
}

func SetupRoutes(
	app *fiber.App,
	handler *Handler,
	authMiddleware middleware.AuthMiddleware,
	meter metric.Meter,
	logger *zap.Logger,
) {
	app.Use(fibercors.New(fibercors.Config{
		AllowCredentials: true,
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, traceparent, tracestate, baggage",
		AllowOriginsFunc: func(origin string) bool {
			return isOriginAllowed(origin, handler.cfg.CORSAllowedOrigins)
		},
	}))
	app.Use(otelfiber.Middleware(otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
		return ctx.Method() + " " + routePattern(ctx)
	})))
	app.Use(middleware.RequestLogger(logger.With(zap.String("component", "http"))))
	app.Use(mustRequestMetricsMiddleware(meter, logger))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(HealthResponse{
			Status:  "ok",
			Service: handler.cfg.AppName,
			Version: handler.cfg.AppVersion,
			Env:     handler.cfg.AppEnv,
		})
	})
	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(VersionResponse{
			Service: handler.cfg.AppName,
			Version: handler.cfg.AppVersion,
			Env:     handler.cfg.AppEnv,
		})
	})

	if handler.cfg.AppEnv != "production" {
		app.Get("/swagger/openapi.yaml", handler.handleOpenAPISpec)
		app.Get("/swagger", handler.handleSwaggerUI)
	}

	api := app.Group("/api")
	if handler.cfg.Auth.Enabled {
		api.Use(authMiddleware.AuthMiddleware())
		api.Get("/session/me", middleware.TraceMiddleware(handler.tracer, handler.logger, "handler.load_session"), handler.loadSession)
	}
	api.Post("/todos", middleware.TraceMiddleware(handler.tracer, handler.logger, "handler.create_todo"), handler.createTodo)

	app.Use(func(c *fiber.Ctx) error {
		return writeErrorResponse(c, errRouteNotFound)
	})
}

// loadSession godoc
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
func (h *Handler) loadSession(c *fiber.Ctx) (err error) {
	logger := util.LoggerWithTrace(c.UserContext(), h.logger).With(zap.String("handler", "load_session"))
	logger.Debug("load session handler started")

	identity, ok := middleware.IdentityFromContext(c.UserContext())
	if !ok || strings.TrimSpace(identity.AuthUserID) == "" {
		return writeErrorResponse(c, errMissingIdentity)
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
	return c.JSON(newSessionResponse(response))
}

// createTodo godoc
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
func (h *Handler) createTodo(c *fiber.Ctx) (err error) {
	logger := util.LoggerWithTrace(c.UserContext(), h.logger).With(zap.String("handler", "create_todo"))
	logger.Debug("create todo handler started")

	var request CreateTodoRequest
	if err := c.BodyParser(&request); err != nil {
		return writeErrorResponse(c, errInvalidBody)
	}

	if err := h.validator.Validate(&request); err != nil {
		return writeErrorResponse(c, errValidationError)
	}

	output, err := h.createTodoUseCase.Execute(c.UserContext(), usecase.CreateTodoInput{
		Title: request.Title,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	logger.Debug("create todo handler completed", zap.String("todo_id", output.Todo.ID))
	return c.Status(fiber.StatusCreated).JSON(newTodoResponse(output.Todo))
}

func (h *Handler) handleOpenAPISpec(c *fiber.Ctx) error {
	if h.cfg.AppEnv == "production" {
		return writeErrorResponse(c, errRouteNotFound)
	}

	payload, err := os.ReadFile("./docs/openapi.yaml")
	if err != nil {
		return err
	}

	c.Set("Content-Type", "application/yaml; charset=utf-8")
	return c.Send(payload)
}

func (h *Handler) handleSwaggerUI(c *fiber.Ctx) error {
	if h.cfg.AppEnv == "production" {
		return writeErrorResponse(c, errRouteNotFound)
	}

	return c.Type("html").SendString(`<!doctype html>
<html lang="ru">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Endge Service Template Scalar</title>
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/swagger/openapi.yaml"
      data-configuration='{"theme":"blue","layout":"modern","showSidebar":true,"persistAuth":true,"defaultOpenAllTags":false}'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.5"></script>
  </body>
</html>`)
}

func mustRequestMetricsMiddleware(meter metric.Meter, logger *zap.Logger) fiber.Handler {
	handler, err := middleware.NewRequestMetricsMiddleware(meter)
	if err != nil {
		logger.Fatal("failed to create request metrics middleware", zap.Error(err))
	}

	return handler
}

func routePattern(c *fiber.Ctx) string {
	if route := c.Route(); route != nil && strings.TrimSpace(route.Path) != "" {
		return route.Path
	}

	return c.Path()
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
	logger := util.LoggerWithTrace(c.UserContext(), h.logger)
	if domainerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
		logger.Error("unexpected request error", fields...)
	} else {
		logger.Warn("request completed with business error", fields...)
	}

	return writeErrorResponse(c, err)
}

func isOriginAllowed(origin string, allowList string) bool {
	normalizedOrigin := strings.TrimSpace(origin)
	if normalizedOrigin == "" {
		return true
	}

	for _, item := range strings.Split(allowList, ",") {
		pattern := strings.TrimSpace(item)
		if pattern == "" {
			continue
		}
		if !strings.Contains(pattern, "*") && strings.EqualFold(pattern, normalizedOrigin) {
			return true
		}
		if strings.HasPrefix(pattern, "https://*.") {
			suffix := strings.TrimPrefix(pattern, "https://*")
			if strings.HasPrefix(normalizedOrigin, "https://") && strings.HasSuffix(normalizedOrigin, suffix) {
				return true
			}
		}
		if strings.HasPrefix(pattern, "http://*.") {
			suffix := strings.TrimPrefix(pattern, "http://*")
			if strings.HasPrefix(normalizedOrigin, "http://") && strings.HasSuffix(normalizedOrigin, suffix) {
				return true
			}
		}
	}

	return false
}
