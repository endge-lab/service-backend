//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestReleaseReadHTTPContracts verifies release list/get authorization and that
// release metadata cannot cross a workspace boundary.
func TestReleaseReadHTTPContracts(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	identity := createReleaseForRead(t, fixture)
	baseURL := "/api/v1/releases"

	unauthenticatedHeaders := map[string]string{"X-Endge-Workspace": fixture.workspace}
	unauthenticatedList := perform(t, fixture.app, http.MethodGet, baseURL, nil, unauthenticatedHeaders)
	assertStatus(t, unauthenticatedList, fiber.StatusUnauthorized)
	unauthenticatedList.Body.Close()
	unauthenticatedGet := perform(t, fixture.app, http.MethodGet, baseURL+"/"+identity, nil, unauthenticatedHeaders)
	assertStatus(t, unauthenticatedGet, fiber.StatusUnauthorized)
	unauthenticatedGet.Body.Close()

	viewerList := perform(t, fixture.app, http.MethodGet, baseURL, nil, fixture.viewer)
	assertStatus(t, viewerList, fiber.StatusOK)
	if !hasIdentity(t, listItems(t, decodeObject(t, viewerList)), identity) {
		t.Fatal("workspace viewer cannot find the release in its list")
	}
	viewerGet := perform(t, fixture.app, http.MethodGet, baseURL+"/"+identity, nil, fixture.viewer)
	assertStatus(t, viewerGet, fiber.StatusOK)
	if stringField(t, decodeObject(t, viewerGet), "identity") != identity {
		t.Fatal("release get returned an unexpected identity")
	}

	outsiderList := perform(t, fixture.app, http.MethodGet, baseURL, nil, fixture.outsider)
	assertStatus(t, outsiderList, fiber.StatusForbidden)
	outsiderList.Body.Close()
	outsiderGet := perform(t, fixture.app, http.MethodGet, baseURL+"/"+identity, nil, fixture.outsider)
	assertStatus(t, outsiderGet, fiber.StatusForbidden)
	outsiderGet.Body.Close()

	missing := perform(t, fixture.app, http.MethodGet, baseURL+"/missing-release", nil, fixture.viewer)
	assertStatus(t, missing, fiber.StatusNotFound)
	missing.Body.Close()
	isolationHeaders := cloneHeaders(fixture.platform)
	isolationHeaders["X-Endge-Workspace"] = fixture.isolatedWorkspace
	isolation := perform(t, fixture.app, http.MethodGet, baseURL+"/"+identity, nil, isolationHeaders)
	assertStatus(t, isolation, fiber.StatusNotFound)
	isolation.Body.Close()
}

func createReleaseForRead(t *testing.T, fixture *serviceAPIFixture) string {
	t.Helper()

	created := perform(t, fixture.app, http.MethodPost, "/api/v1/queries", map[string]any{
		"identity": "release-read-query", "displayName": "Release read query", "source": "query {}", "sourceVersion": 2,
	}, fixture.workspaceAdmin)
	assertStatus(t, created, fiber.StatusCreated)
	created.Body.Close()
	commit := perform(t, fixture.app, http.MethodPost, "/api/v1/commits", map[string]any{
		"message": "Release read baseline", "revisionPolicy": "preserve",
		"expectedHeadSequence": currentHeadSequence(t, fixture.app, fixture.workspaceAdmin),
	}, fixture.workspaceAdmin)
	assertStatus(t, commit, fiber.StatusCreated)
	commitID := stringField(t, decodeObject(t, commit), "id")

	release := perform(t, fixture.app, http.MethodPost, "/api/v1/releases", map[string]any{
		"identity": "release-read", "displayName": "Release read", "sourceCommitId": commitID,
	}, fixture.workspaceAdmin)
	assertStatus(t, release, fiber.StatusCreated)
	return stringField(t, decodeObject(t, release), "identity")
}
