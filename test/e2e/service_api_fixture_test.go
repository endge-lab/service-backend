//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// serviceAPIFixture creates a stable role matrix for service API E2E tests.
// The viewer is a member only of workspace; isolatedWorkspace remains visible
// solely to the platform administrator and is used to verify isolation.
type serviceAPIFixture struct {
	app               *fiber.App
	platform          map[string]string
	workspaceAdmin    map[string]string
	viewer            map[string]string
	outsider          map[string]string
	outsiderID        string
	workspace         string
	isolatedWorkspace string
}

func newServiceAPIFixture(t *testing.T) *serviceAPIFixture {
	t.Helper()

	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))

	platform := bearer(provider.Token(t, support.TokenInput{
		Subject: "service-api-platform", Username: "service-api-platform", DisplayName: "Service API Platform", Groups: []string{"endge-platform-admins"},
	}))
	admin := bearer(provider.Token(t, support.TokenInput{
		Subject: "service-api-admin", Username: "service-api-admin", DisplayName: "Service API Admin",
	}))
	viewer := bearer(provider.Token(t, support.TokenInput{
		Subject: "service-api-viewer", Username: "service-api-viewer", DisplayName: "Service API Viewer",
	}))
	outsider := bearer(provider.Token(t, support.TokenInput{
		Subject: "service-api-outsider", Username: "service-api-outsider", DisplayName: "Service API Outsider",
	}))

	// Register the configured platform administrator first. Otherwise the
	// bootstrap rule promotes the first ordinary user to platform admin too.
	_ = currentUserID(t, app, platform)
	adminID := currentUserID(t, app, admin)
	viewerID := currentUserID(t, app, viewer)
	outsiderID := currentUserID(t, app, outsider)

	workspace := "service-api-workspace"
	createWorkspace(t, app, platform, workspace)
	putMembership(t, app, platform, workspace, adminID, "admin")
	putMembership(t, app, platform, workspace, viewerID, "viewer")

	isolatedWorkspace := "service-api-isolated"
	createWorkspace(t, app, platform, isolatedWorkspace)

	return &serviceAPIFixture{
		app:               app,
		platform:          workspaceHeaders(platform, workspace),
		workspaceAdmin:    workspaceHeaders(admin, workspace),
		viewer:            workspaceHeaders(viewer, workspace),
		outsider:          workspaceHeaders(outsider, workspace),
		outsiderID:        outsiderID,
		workspace:         workspace,
		isolatedWorkspace: isolatedWorkspace,
	}
}

// TestServiceAPIFixtureRoleMatrix prevents later service API tests from using
// an invalid role fixture and accidentally asserting the wrong authorization.
func TestServiceAPIFixtureRoleMatrix(t *testing.T) {
	fixture := newServiceAPIFixture(t)

	assertSessionWorkspaceRole(t, fixture.app, fixture.platform, fixture.workspace, "admin")
	assertSessionWorkspaceRole(t, fixture.app, fixture.workspaceAdmin, fixture.workspace, "admin")
	assertSessionWorkspaceRole(t, fixture.app, fixture.viewer, fixture.workspace, "viewer")

	response := perform(t, fixture.app, http.MethodGet, "/api/v1/queries", nil, fixture.outsider)
	assertStatus(t, response, fiber.StatusForbidden)
	response.Body.Close()
}

func createWorkspace(t *testing.T, app *fiber.App, headers map[string]string, identity string) {
	t.Helper()

	response := perform(t, app, http.MethodPost, "/api/v1/workspaces", map[string]any{
		"identity": identity, "displayName": identity, "dataMode": "development",
	}, headers)
	assertStatus(t, response, fiber.StatusCreated)
	response.Body.Close()
}
