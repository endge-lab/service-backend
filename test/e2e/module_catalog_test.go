//go:build e2e

package e2e_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGlobalModuleCatalog(t *testing.T) {
	fixture := newServiceAPIFixture(t)
	app := fixture.app
	path := "/api/v1/modules/slot-ui"
	platform := cloneHeaders(fixture.platform)
	delete(platform, "X-Endge-Workspace")
	viewer := cloneHeaders(fixture.viewer)
	delete(viewer, "X-Endge-Workspace")
	fields := map[string]string{"name": "Slot UI"}
	original := moduleZIP(t, "0.1.0-dev.1+build", "export default () => ({})")

	assertStatus(t, perform(t, app, http.MethodGet, "/api/v1/modules", nil, nil), http.StatusUnauthorized)
	assertStatus(t, uploadModule(t, app, "/api/v1/modules", viewer, fields, original), http.StatusForbidden)
	assertStatus(t, uploadModule(t, app, "/api/v1/modules", platform, fields, []byte("invalid")), http.StatusBadRequest)
	assertStatus(t, uploadModule(t, app, "/api/v1/modules", platform, map[string]string{
		"name": "Slot UI", "identity": "another-module",
	}, original), http.StatusBadRequest)
	assertStatus(t, perform(t, app, http.MethodGet, path, nil, viewer), http.StatusNotFound)

	created := uploadModule(t, app, "/api/v1/modules", platform, fields, original)
	assertStatus(t, created, http.StatusCreated)
	if etag := created.Header.Get("ETag"); etag != "\"1\"" {
		t.Fatalf("ETag: %s", etag)
	}
	moduleID := stringField(t, decodeObject(t, created), "id")
	assertStatus(t, uploadModule(t, app, "/api/v1/modules", platform, fields, original), http.StatusConflict)
	for _, workspace := range []string{fixture.workspace, fixture.isolatedWorkspace} {
		response := perform(t, app, http.MethodGet, path, nil, workspaceHeaders(viewer, workspace))
		assertStatus(t, response, http.StatusOK)
		if identity := stringField(t, decodeObject(t, response), "identity"); identity != "slot-ui" {
			t.Fatalf("module not visible globally: %s", identity)
		}
	}

	versionPath := path + "/versions/0.1.0-dev.1%2Bbuild"
	response := perform(t, app, http.MethodGet, versionPath, nil, viewer)
	assertStatus(t, response, http.StatusOK)
	manifest := decodeObject(t, response)
	archiveURL := stringField(t, manifest, "archiveUrl")
	payload := perform(t, app, http.MethodGet, archiveURL, nil, viewer)
	assertStatus(t, payload, http.StatusOK)
	defer payload.Body.Close()
	saved, err := io.ReadAll(payload.Body)
	if err != nil || !bytes.Equal(saved, original) {
		t.Fatal("download must return the original ZIP")
	}
	assets := objectField(t, manifest, "assets")
	js := stringField(t, assets, "js")
	assertStatus(t, perform(t, app, http.MethodGet, js, nil, nil), http.StatusUnauthorized)
	javascript := perform(t, app, http.MethodGet, js, nil, viewer)
	assertStatus(t, javascript, http.StatusOK)
	if javascript.Header.Get("Content-Type") != "text/javascript; charset=utf-8" {
		t.Fatal("incorrect module.js content type")
	}
	assertStatus(t, perform(t, app, http.MethodGet, stringField(t, assets, "css"), nil, viewer), http.StatusOK)

	versions := path + "/versions"
	assertStatus(t, uploadModule(t, app, versions, platform, nil, original), http.StatusCreated)
	conflict := moduleZIP(t, "0.1.0-dev.1+build", "export default () => undefined")
	assertStatus(t, uploadModule(t, app, versions, platform, nil, conflict), http.StatusConflict)
	next := moduleZIP(t, "0.1.0-dev.2", "export default () => ({})")
	assertStatus(t, uploadModule(t, app, versions, platform, nil, next), http.StatusCreated)
	assertStatus(t, perform(t, app, http.MethodPatch, path, map[string]any{"name": "New name"}, platform), http.StatusPreconditionRequired)
	patch := cloneHeaders(platform)
	patch["If-Match"] = created.Header.Get("ETag")
	updated := perform(t, app, http.MethodPatch, path, map[string]any{"name": "New name"}, patch)
	assertStatus(t, updated, http.StatusOK)
	assertStatus(t, perform(t, app, http.MethodPatch, path, map[string]any{"name": "Old name"}, patch), http.StatusConflict)

	archivePath := "/api/v1/modules?deleted=true"
	archiveHeaders := cloneHeaders(platform)
	archiveHeaders["If-Match"] = updated.Header.Get("ETag")
	viewerMatch := cloneHeaders(viewer)
	viewerMatch["If-Match"] = archiveHeaders["If-Match"]
	assertStatus(t, perform(t, app, http.MethodDelete, path, nil, viewerMatch), http.StatusForbidden)
	assertStatus(t, perform(t, app, http.MethodGet, archivePath, nil, viewer), http.StatusForbidden)
	assertStatus(t, perform(t, app, http.MethodDelete, path, nil, patch), http.StatusConflict)

	archived := perform(t, app, http.MethodDelete, path, nil, archiveHeaders)
	assertStatus(t, archived, http.StatusOK)
	if stringField(t, decodeObject(t, archived), "deletedAt") == "" {
		t.Fatal("archiving must record a timestamp")
	}
	assertStatus(t, perform(t, app, http.MethodGet, path, nil, viewer), http.StatusNotFound)
	activeList := perform(t, app, http.MethodGet, "/api/v1/modules", nil, viewer)
	assertStatus(t, activeList, http.StatusOK)
	if numberField(t, decodeObject(t, activeList), "total") != 0 {
		t.Fatal("archived module must be absent from the active catalog")
	}
	archivedList := perform(t, app, http.MethodGet, archivePath, nil, platform)
	assertStatus(t, archivedList, http.StatusOK)
	if numberField(t, decodeObject(t, archivedList), "total") != 1 {
		t.Fatal("archived module must be visible to an administrator")
	}
	assertStatus(t, uploadModule(t, app, versions, platform, nil, next), http.StatusNotFound)
	assertStatus(t, perform(t, app, http.MethodGet, versionPath, nil, viewer), http.StatusOK)
	assertStatus(t, perform(t, app, http.MethodGet, js, nil, viewer), http.StatusOK)

	replacementFields := map[string]string{"name": "Restored Slot UI", "description": "Updated renderers"}
	assertStatus(t, uploadModule(t, app, "/api/v1/modules", platform, replacementFields, conflict), http.StatusConflict)
	assertStatus(t, perform(t, app, http.MethodGet, path, nil, viewer), http.StatusNotFound)

	restorePath := path + "/restore"
	assertStatus(t, perform(t, app, http.MethodPost, restorePath, nil, archiveHeaders), http.StatusConflict)
	archiveHeaders["If-Match"] = archived.Header.Get("ETag")
	assertStatus(t, perform(t, app, http.MethodPost, restorePath, nil, archiveHeaders), http.StatusOK)
	restored := perform(t, app, http.MethodGet, path, nil, viewer)
	assertStatus(t, restored, http.StatusOK)
	if versions := arrayField(t, decodeObject(t, restored), "versions"); len(versions) != 2 {
		t.Fatalf("restore must retain published versions, got %d", len(versions))
	}

	archiveHeaders["If-Match"] = restored.Header.Get("ETag")
	assertStatus(t, perform(t, app, http.MethodDelete, path, nil, archiveHeaders), http.StatusOK)
	// Re-uploading the identical ZIP restores the existing card and reuses its published version.
	reuploaded := uploadModule(t, app, "/api/v1/modules", platform, replacementFields, original)
	assertStatus(t, reuploaded, http.StatusCreated)
	reuploadedModule := decodeObject(t, reuploaded)
	if stringField(t, reuploadedModule, "id") != moduleID ||
		stringField(t, reuploadedModule, "name") != replacementFields["name"] ||
		stringField(t, reuploadedModule, "description") != replacementFields["description"] ||
		len(arrayField(t, reuploadedModule, "versions")) != 2 {
		t.Fatal("re-upload must restore the same module with the submitted metadata and both existing versions")
	}

	archiveHeaders["If-Match"] = reuploaded.Header.Get("ETag")
	assertStatus(t, perform(t, app, http.MethodDelete, path, nil, archiveHeaders), http.StatusOK)
	newVersion := moduleZIP(t, "0.1.0-dev.3", "export default () => ({})")
	recreated := uploadModule(t, app, "/api/v1/modules", platform, replacementFields, newVersion)
	assertStatus(t, recreated, http.StatusCreated)
	if versions := arrayField(t, decodeObject(t, recreated), "versions"); len(versions) != 3 {
		t.Fatalf("upload with a new version must restore the module and publish the ZIP, got %d versions", len(versions))
	}
	assertStatus(t, perform(t, app, http.MethodGet, path+"/versions/0.1.0-dev.3", nil, viewer), http.StatusOK)
}

func moduleZIP(t *testing.T, version, source string) []byte {
	t.Helper()
	var payload bytes.Buffer
	writer := zip.NewWriter(&payload)
	for _, file := range []struct{ name, source string }{
		{"manifest.json", fmt.Sprintf(`{"identity":"slot-ui","version":%q,"formatVersion":1,"hostApi":1}`, version)},
		{"module.js", source},
		{"module.css", ".slot-table{color:#297bba}"},
	} {
		w, err := writer.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(w, file.source); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return payload.Bytes()
}

func uploadModule(t *testing.T, app *fiber.App, path string, headers, fields map[string]string, data []byte) *http.Response {
	t.Helper()
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.CreateFormFile("archive", "module.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, &payload)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
