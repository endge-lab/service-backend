//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestBuildProfileHTTPContractsAndRoleMatrix(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	provider := support.NewIdentityProvider(t)
	app := support.NewTestApp(t, database, support.OIDCConfig(provider))

	platform := bearer(provider.Token(t, support.TokenInput{
		Subject: "profile-platform", Username: "profile-platform", DisplayName: "Profile Platform", Groups: []string{"endge-platform-admins"},
	}))
	owner := bearer(provider.Token(t, support.TokenInput{Subject: "profile-owner", Username: "profile-owner", DisplayName: "Profile Owner"}))
	other := bearer(provider.Token(t, support.TokenInput{Subject: "profile-other", Username: "profile-other", DisplayName: "Profile Other"}))
	viewer := bearer(provider.Token(t, support.TokenInput{Subject: "profile-viewer", Username: "profile-viewer", DisplayName: "Profile Viewer"}))

	_ = currentUserID(t, app, platform)
	ownerID := currentUserID(t, app, owner)
	otherID := currentUserID(t, app, other)
	viewerID := currentUserID(t, app, viewer)
	createWorkspace(t, app, platform, "profile-workspace")
	putMembership(t, app, platform, "profile-workspace", ownerID, "editor")
	putMembership(t, app, platform, "profile-workspace", otherID, "editor")
	putMembership(t, app, platform, "profile-workspace", viewerID, "viewer")

	ownerHeaders := workspaceHeaders(owner, "profile-workspace")
	otherHeaders := workspaceHeaders(other, "profile-workspace")
	viewerHeaders := workspaceHeaders(viewer, "profile-workspace")
	endpoint := "/api/v1/build-profiles"

	viewerCreate := perform(t, app, http.MethodPost, endpoint, buildProfilePayload("private"), viewerHeaders)
	assertStatus(t, viewerCreate, fiber.StatusForbidden)
	viewerCreate.Body.Close()

	privateResponse := perform(t, app, http.MethodPost, endpoint, buildProfilePayload("private"), ownerHeaders)
	assertStatus(t, privateResponse, fiber.StatusCreated)
	privateETag := privateResponse.Header.Get("ETag")
	privateProfile := decodeObject(t, privateResponse)
	privateIdentity := stringField(t, privateProfile, "identity")
	privateID := stringField(t, privateProfile, "id")
	if _, err := uuid.Parse(privateID); err != nil {
		t.Fatalf("profile id is not UUID: %q", privateID)
	}
	if _, err := uuid.Parse(privateIdentity); err != nil || privateIdentity == privateID {
		t.Fatalf("profile identity must be an independent UUID: id=%q identity=%q", privateID, privateIdentity)
	}
	if stringField(t, privateProfile, "displayName") != "Новый профиль 1" || privateETag != `"1"` {
		t.Fatalf("unexpected first profile: etag=%q body=%#v", privateETag, privateProfile)
	}
	if privateProfile["ownedByMe"] != true || privateProfile["canManage"] != true || privateProfile["canChangeVisibility"] != true {
		t.Fatalf("owner capabilities are incorrect: %#v", privateProfile)
	}

	sharedResponse := perform(t, app, http.MethodPost, endpoint, buildProfilePayload("shared"), ownerHeaders)
	assertStatus(t, sharedResponse, fiber.StatusCreated)
	sharedProfile := decodeObject(t, sharedResponse)
	sharedIdentity := stringField(t, sharedProfile, "identity")
	if stringField(t, sharedProfile, "displayName") != "Новый профиль 2" {
		t.Fatalf("automatic numbering is incorrect: %#v", sharedProfile)
	}

	viewerItems := listItems(t, decodeObject(t, perform(t, app, http.MethodGet, endpoint, nil, viewerHeaders)))
	if len(viewerItems) != 1 {
		t.Fatalf("viewer did not receive exactly one shared profile: %#v", viewerItems)
	}
	viewerProfile, _ := viewerItems[0].(map[string]any)
	if stringField(t, viewerProfile, "identity") != sharedIdentity {
		t.Fatalf("viewer did not receive exactly the shared profile: %#v", viewerItems)
	}
	if viewerProfile["canManage"] != false || viewerProfile["canChangeVisibility"] != false {
		t.Fatalf("viewer received write capabilities: %#v", viewerProfile)
	}
	otherItems := listItems(t, decodeObject(t, perform(t, app, http.MethodGet, endpoint, nil, otherHeaders)))
	if len(otherItems) != 1 {
		t.Fatalf("other editor did not receive exactly one shared profile: %#v", otherItems)
	}
	otherProfile, _ := otherItems[0].(map[string]any)
	if otherProfile["canManage"] != true || otherProfile["canChangeVisibility"] != false {
		t.Fatalf("other editor shared projection is incorrect: %#v", otherItems)
	}
	ownerItems := listItems(t, decodeObject(t, perform(t, app, http.MethodGet, endpoint, nil, ownerHeaders)))
	if len(ownerItems) != 2 {
		t.Fatalf("owner must see shared and own private profiles: %#v", ownerItems)
	}

	missingIfMatch := perform(t, app, http.MethodPatch, endpoint+"/"+sharedIdentity, map[string]any{"displayName": "Missing"}, otherHeaders)
	assertStatus(t, missingIfMatch, fiber.StatusPreconditionRequired)
	missingIfMatch.Body.Close()
	patchHeaders := cloneHeaders(otherHeaders)
	patchHeaders["If-Match"] = `"1"`
	patchedResponse := perform(t, app, http.MethodPatch, endpoint+"/"+sharedIdentity, map[string]any{"displayName": "  Общий профиль  "}, patchHeaders)
	assertStatus(t, patchedResponse, fiber.StatusOK)
	patched := decodeObject(t, patchedResponse)
	if stringField(t, patched, "displayName") != "Общий профиль" || numberField(t, patched, "revision") != 2 {
		t.Fatalf("shared patch result is incorrect: %#v", patched)
	}
	stale := perform(t, app, http.MethodPatch, endpoint+"/"+sharedIdentity, map[string]any{"displayName": "Stale"}, patchHeaders)
	assertStatus(t, stale, fiber.StatusConflict)
	stale.Body.Close()

	makePrivate := perform(t, app, http.MethodPatch, endpoint+"/"+sharedIdentity, map[string]any{"visibility": "private"}, withIfMatch(otherHeaders, `"2"`))
	assertStatus(t, makePrivate, fiber.StatusForbidden)
	makePrivate.Body.Close()
	ownerPrivate := perform(t, app, http.MethodPatch, endpoint+"/"+sharedIdentity, map[string]any{"visibility": "private"}, withIfMatch(ownerHeaders, `"2"`))
	assertStatus(t, ownerPrivate, fiber.StatusOK)
	ownerPrivate.Body.Close()

	deleteOtherPrivate := perform(t, app, http.MethodDelete, endpoint+"/"+privateIdentity, nil, withIfMatch(otherHeaders, privateETag))
	assertStatus(t, deleteOtherPrivate, fiber.StatusForbidden)
	deleteOtherPrivate.Body.Close()
	deleteOwnerPrivate := perform(t, app, http.MethodDelete, endpoint+"/"+privateIdentity, nil, withIfMatch(ownerHeaders, privateETag))
	assertStatus(t, deleteOwnerPrivate, fiber.StatusNoContent)
	deleteOwnerPrivate.Body.Close()

	remaining := listItems(t, decodeObject(t, perform(t, app, http.MethodGet, endpoint, nil, ownerHeaders)))
	if len(remaining) != 1 {
		t.Fatalf("physical delete left unexpected rows: %#v", remaining)
	}
	remainingProfile, _ := remaining[0].(map[string]any)
	if stringField(t, remainingProfile, "identity") != sharedIdentity {
		t.Fatalf("physical delete left unexpected rows: %#v", remaining)
	}

	const parallelCreates = 6
	encodedPayload, err := json.Marshal(buildProfilePayload("private"))
	if err != nil {
		t.Fatalf("encode concurrent payload: %v", err)
	}
	type createResult struct {
		status int
		name   string
		err    error
	}
	results := make(chan createResult, parallelCreates)
	for range parallelCreates {
		go func() {
			request := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(encodedPayload))
			request.Header.Set("Content-Type", "application/json")
			for name, value := range ownerHeaders {
				request.Header.Set(name, value)
			}
			response, requestErr := app.Test(request, -1)
			if requestErr != nil {
				results <- createResult{err: requestErr}
				return
			}
			defer response.Body.Close()
			var body map[string]any
			decodeErr := json.NewDecoder(response.Body).Decode(&body)
			results <- createResult{status: response.StatusCode, name: fmt.Sprint(body["displayName"]), err: decodeErr}
		}()
	}
	names := map[string]bool{}
	for range parallelCreates {
		result := <-results
		if result.err != nil || result.status != fiber.StatusCreated {
			t.Fatalf("concurrent create status=%d name=%q err=%v", result.status, result.name, result.err)
		}
		if names[result.name] {
			t.Fatalf("concurrent auto-name was duplicated: %q", result.name)
		}
		names[result.name] = true
	}
}

func buildProfilePayload(visibility string) map[string]any {
	return map[string]any{
		"visibility": visibility,
		"settings": map[string]any{
			"buildScope": "complete-model", "contexts": "all-contexts", "diagnostics": "detailed",
			"debuggerStructure": "complete-catalog",
			"topology":          []any{map[string]any{"node": "frontend", "runtime": "ts-browser"}},
		},
	}
}

func withIfMatch(headers map[string]string, etag string) map[string]string {
	result := cloneHeaders(headers)
	result["If-Match"] = etag
	return result
}
