package http

import (
	"strings"
	"testing"

	httpmiddleware "github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/gofiber/fiber/v2"
)

// tokenAllowedUnscopedRoutes — маршруты /api вне workspace scope, которые персональный токен
// может вызывать. Новый маршрут вне workspace должен попасть либо сюда (осознанно), либо
// в закрытые разделы политики токенов.
var tokenAllowedUnscopedRoutes = map[string]bool{
	"GET /api/session/me": true,
}

// TestAccessTokenPolicyCoversUnscopedRoutes сверяет все маршруты /api вне workspace scope
// с политикой персональных токенов: роль workspace их не ограничивает, поэтому открытым
// остаётся только явно перечисленное.
func TestAccessTokenPolicyCoversUnscopedRoutes(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api")
	registerUnscopedAPIRoutes(api, Handlers{})

	checked := 0
	for _, route := range app.GetRoutes(true) {
		if route.Method == fiber.MethodHead || route.Method == fiber.MethodOptions || !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		key := route.Method + " " + route.Path
		checked++
		allowed := httpmiddleware.AccessTokenAllowsRoute(route.Method, strings.TrimPrefix(route.Path, "/api"))
		if allowed && !tokenAllowedUnscopedRoutes[key] {
			t.Errorf("unscoped route %s is open to access tokens: add it to the denied sections or to tokenAllowedUnscopedRoutes", key)
		}
		if !allowed && tokenAllowedUnscopedRoutes[key] {
			t.Errorf("route %s is listed as allowed for tokens but the policy denies it", key)
		}
	}
	if checked < 20 {
		t.Fatalf("expected the unscoped API surface to be registered, checked only %d routes", checked)
	}
}
