package mcp

import (
	"context"
	"net/http"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// requestContextKey переносит context запроса Fiber (identity, токен, actor) в net/http-запрос MCP:
// стандартный адаптер Fiber строит context из fasthttp и не видит UserContext.
type requestContextKey struct{}

// Handler обслуживает MCP Streamable HTTP на {HTTP_BASE_PATH}/mcp.
//
// Сервер stateless: Mcp-Session-Id не выдаётся, каждый POST обрабатывается временной
// сессией, поэтому запросы можно направлять на любой экземпляр за балансировщиком.
// Ответы — application/json без SSE. GET и DELETE отвечают 405, как разрешает спецификация.
type Handler struct {
	servers    map[permissions]*mcp.Server
	workspaces WorkspaceAuthorizer
	serve      fiber.Handler
}

// NewHandler собирает MCP-серверы для всех наборов прав и net/http-адаптер.
func NewHandler(cfg Config, useCases UseCases, contracts Contracts, logger *zap.Logger) *Handler {
	h := &Handler{servers: map[permissions]*mcp.Server{}, workspaces: useCases.Workspaces}
	// Один источник документации на все наборы прав: общий кеш файлов.
	docs := newDocsSource(cfg.AgentDocsURL, cfg.Version)
	for _, write := range []bool{false, true} {
		for _, commit := range []bool{false, true} {
			for _, remove := range []bool{false, true} {
				for _, token := range []bool{false, true} {
					set := permissions{write: write, commit: commit, delete: remove, token: token}
					h.servers[set] = newServer(cfg, useCases, contracts, logger, set, docs)
				}
			}
		}
	}
	streamable := mcp.NewStreamableHTTPHandler(h.server, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	h.serve = adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := r.Context().Value(requestContextKey{}).(context.Context); ok {
			r = r.WithContext(ctx)
		}
		streamable.ServeHTTP(w, r)
	}))
	return h
}

// server выбирает сервер с инструментами, которые разрешает токен или сессия запроса.
func (h *Handler) server(r *http.Request) *mcp.Server {
	allowed := permissionsFrom(r.Context())
	if allowed.commit && !h.commitAvailable(r.Context()) {
		allowed.commit = false
	}
	return h.servers[allowed]
}

// commitAvailable скрывает endge_commit, если коммит заведомо будет отклонён use case.
// Токен с флагом commit: владелец должен быть Admin workspace токена или Platform Admin
// (access_token_owner_admin_required). Сессия пользователя выбирает workspace в каждом
// вызове, поэтому инструмент показывается Platform Admin и пользователю с ролью Admin
// хотя бы в одном workspace; роль в конкретном workspace по-прежнему проверяет use case
// (workspace_admin_required).
func (h *Handler) commitAvailable(ctx context.Context) bool {
	if token, ok := entities.AccessTokenFromContext(ctx); ok {
		access, err := h.workspaces.Authorize(ctx, token.WorkspaceIdentity)
		return err == nil && shared.CanAdmin(access.OwnerRole)
	}
	if actor, ok := entities.CurrentActorFromContext(ctx); ok && actor.PlatformAdmin {
		return true
	}
	items, err := h.workspaces.ListAccess(ctx)
	if err != nil {
		return false
	}
	for _, access := range items {
		if shared.CanAdmin(access.Role) {
			return true
		}
	}
	return false
}

// Serve обрабатывает сообщение MCP.
// @Summary MCP: сообщение клиента
// @Description MCP-сервер Endge (Streamable HTTP, stateless, ответы application/json). Принимает JSON-RPC сообщения MCP: initialize, tools/list, tools/call, resources/list, resources/templates/list, resources/read. Инструменты вызывают те же use case, что REST; персональный токен ограничивает workspace, набор инструментов (scope, флаги commit и delete) и помечает изменения в истории ревизий. Accept должен содержать application/json и text/event-stream.
// @ID postMCP
// @Tags MCP
// @Accept json
// @Produce json
// @Param request body JSONRPCRequest true "Сообщение JSON-RPC 2.0"
// @Success 200 {object} JSONRPCResponse "Ответ JSON-RPC"
// @Success 202 "Уведомление принято"
// @Failure 400 {string} string "Некорректное сообщение или заголовок Accept"
// @Failure 401 {object} shared.ErrorResponse "Требуется персональный токен или JWT; заголовок WWW-Authenticate: Bearer"
// @Failure 403 {object} shared.ErrorResponse "Пользователь неактивен"
// @Failure 415 {string} string "Content-Type должен быть application/json"
// @Security BearerAuth
// @Router /mcp [post]
func (h *Handler) Serve(c *fiber.Ctx) error {
	c.Context().SetUserValue(requestContextKey{}, c.UserContext())
	return h.serve(c)
}

// OpenStream отвечает на GET /mcp: stateless-сервер не держит SSE-поток.
// @Summary MCP: поток уведомлений сервера
// @Description Stateless MCP-сервер не открывает SSE-поток и отвечает 405 с Allow: POST, как разрешает спецификация Streamable HTTP.
// @ID getMCP
// @Tags MCP
// @Produce json
// @Failure 401 {object} shared.ErrorResponse "Требуется персональный токен или JWT"
// @Failure 405 {string} string "Поток не поддерживается"
// @Security BearerAuth
// @Router /mcp [get]
func (h *Handler) OpenStream(c *fiber.Ctx) error { return h.Serve(c) }

// CloseSession отвечает на DELETE /mcp: сессий нет, закрывать нечего.
// @Summary MCP: завершение сессии
// @Description Stateless MCP-сервер не выдаёт Mcp-Session-Id и отвечает 405 с Allow: POST.
// @ID deleteMCP
// @Tags MCP
// @Produce json
// @Failure 401 {object} shared.ErrorResponse "Требуется персональный токен или JWT"
// @Failure 405 {string} string "Сессии не поддерживаются"
// @Security BearerAuth
// @Router /mcp [delete]
func (h *Handler) CloseSession(c *fiber.Ctx) error { return h.Serve(c) }

// Challenge добавляет WWW-Authenticate к 401 от auth middleware: MCP-клиенты по нему
// понимают, что нужен Bearer (персональный токен Endge или JWT).
func Challenge() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if c.Response().StatusCode() != fiber.StatusUnauthorized {
			return err
		}
		// По RFC 6750 код invalid_token указывается, только если клиент передал учётные данные.
		challenge := `Bearer realm="endge"`
		if c.Get(fiber.HeaderAuthorization) != "" {
			challenge += `, error="invalid_token"`
		}
		c.Set(fiber.HeaderWWWAuthenticate, challenge+`, error_description="Use an Endge personal access token (endge_pat_...) or a user JWT"`)
		return err
	}
}

// RegisterRoutes регистрирует POST/GET/DELETE /mcp с теми же auth и current user, что /api.
// Политика токенов /api не нужна: инструменты MCP не обращаются к закрытым разделам,
// а флаги delete/commit и workspace токена проверяются в инструментах и use case.
func RegisterRoutes(router fiber.Router, handler *Handler, auth, currentUser fiber.Handler) {
	router.Post("/mcp", Challenge(), auth, currentUser, handler.Serve)
	router.Get("/mcp", Challenge(), auth, currentUser, handler.OpenStream)
	router.Delete("/mcp", Challenge(), auth, currentUser, handler.CloseSession)
}
