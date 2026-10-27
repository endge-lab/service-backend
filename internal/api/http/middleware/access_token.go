package middleware

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/access_tokens"
	"github.com/gofiber/fiber/v2"
)

const accessTokenOwnerKey contextKey = "access_token_owner"

// accessTokenDeniedPrefixes — разделы /api, закрытые для персональных токенов целиком:
// управление доступом и токенами, workspace CRUD, AI, подключения, интеграции, Bridge,
// импорт и резервные копии. Остальные маршруты дополнительно проверяют роль,
// уже суженную scope токена.
var accessTokenDeniedPrefixes = []string{
	"/v1/access-tokens",
	"/v1/access-grants",
	"/v1/service-users",
	"/v1/workspaces",
	"/v1/ai",
	"/v1/backend-connections",
	"/v1/integrations",
	"/v1/bridge",
	"/v1/domain/import",
	"/v1/domain/backups",
}

// AccessTokenAllowsRoute сообщает, пропускает ли политика токена маршрут /api (путь относительно /api)
// для токена с полными флагами. Нужна архитектурному тесту маршрутов вне workspace.
func AccessTokenAllowsRoute(method, route string) bool {
	full := entities.AccessTokenPrincipal{Scope: entities.AccessTokenScopeWrite, CanCommit: true, CanDelete: true}
	return accessTokenRouteError(full, method, normalizeRoutePath(route, "")) == nil
}

// bearerAccessToken выделяет PAT из заголовка Authorization до проверки JWT.
func bearerAccessToken(header string) (string, bool) {
	if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
		return "", false
	}
	value := strings.TrimSpace(header[7:])
	return value, access_tokens.IsAccessTokenSecret(value)
}

// authenticateAccessToken проверяет PAT и кладёт в context identity владельца и ограничения токена.
// Секрет не попадает в ответ, логи и context.
func (m *authMiddleware) authenticateAccessToken(c *fiber.Ctx, secret string) error {
	result, err := m.tokens.Authenticate(c.UserContext(), secret)
	if err != nil {
		if domainerrors.HTTPStatusOf(err) == fiber.StatusUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": domainerrors.CodeOf(err), "message": domainerrors.SafeMessageOf(err)})
		}
		return respond.RespondDomainError(c, nil, err)
	}
	owner := result.Owner
	identity := RequestIdentity{
		ProviderID: owner.ProviderID, Subject: owner.Subject, Issuer: owner.Issuer, AuthUserID: owner.ID,
		Username: owner.Username, DisplayName: owner.DisplayName, AccessTokenID: result.Principal.TokenID,
		ExpiresAt: result.Principal.ExpiresAt.Format(time.RFC3339),
	}
	ctx := entities.WithAccessToken(c.UserContext(), result.Principal)
	ctx = context.WithValue(ctx, identityKey, identity)
	ctx = context.WithValue(ctx, accessTokenOwnerKey, &owner)
	c.SetUserContext(ctx)
	return c.Next()
}

// AccessTokenPolicy ограничивает запросы с PAT: закрытые разделы, удаление и фиксацию
// изменений по флагам токена. Cookie и JWT запросы проходят без изменений.
func AccessTokenPolicy(basePath string) fiber.Handler {
	apiPrefix := strings.ToLower(strings.TrimRight(basePath, "/")) + "/api"
	return func(c *fiber.Ctx) error {
		token, ok := entities.AccessTokenFromContext(c.UserContext())
		if !ok {
			return c.Next()
		}
		route := normalizeRoutePath(c.Path(), apiPrefix)
		if err := accessTokenRouteError(token, c.Method(), route); err != nil {
			return respond.WriteErrorResponse(c, err)
		}
		return c.Next()
	}
}

// normalizeRoutePath повторяет нечувствительный к регистру и хвостовому слэшу роутинг Fiber,
// чтобы /API/V1/Workspaces/ не обходил запрет.
func normalizeRoutePath(rawPath, apiPrefix string) string {
	cleaned := path.Clean("/" + strings.ToLower(rawPath))
	return strings.TrimPrefix(cleaned, apiPrefix)
}

func accessTokenRouteError(token entities.AccessTokenPrincipal, method, route string) error {
	for _, prefix := range accessTokenDeniedPrefixes {
		if route == prefix || strings.HasPrefix(route, prefix+"/") {
			if prefix == "/v1/access-tokens" {
				return domainerrors.Forbidden("access_token_management_forbidden", "Access tokens can be managed only from a user session")
			}
			return domainerrors.Forbidden("access_token_route_forbidden", "This section is not available to access tokens")
		}
	}
	if isReleaseArtifactRoute(route) {
		return domainerrors.Forbidden("access_token_route_forbidden", "Release bundles and exports are not available to access tokens")
	}
	if method == fiber.MethodDelete && !token.CanDelete {
		return domainerrors.Forbidden("access_token_delete_forbidden", "Access token does not allow deletion")
	}
	if method == fiber.MethodPost && isCommitRoute(route) && !token.CanCommit {
		return domainerrors.Forbidden("access_token_commit_forbidden", "Access token does not allow commits and releases")
	}
	return nil
}

// isCommitRoute — операции, фиксирующие историю: commit, restore commit, release и его restore.
// Планы (/plan) только читают и проверяются ролью.
func isCommitRoute(route string) bool {
	if strings.HasSuffix(route, "/plan") {
		return false
	}
	switch {
	case route == "/v1/commits", route == "/v1/releases", route == "/v1/releases/from-build":
		return true
	case strings.HasPrefix(route, "/v1/commits/") && strings.HasSuffix(route, "/restore"):
		return true
	case strings.HasPrefix(route, "/v1/releases/") && strings.HasSuffix(route, "/restore"):
		return true
	}
	return false
}

// isReleaseArtifactRoute — выгрузка готового релиза (bundle и export): агенту она не нужна,
// а полный артефакт workspace не должен уходить по токену.
func isReleaseArtifactRoute(route string) bool {
	if !strings.HasPrefix(route, "/v1/releases/") {
		return false
	}
	return strings.HasSuffix(route, "/bundle") || strings.HasSuffix(route, "/export")
}
