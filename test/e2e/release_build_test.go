//go:build e2e

package e2e_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

func TestBuiltReleaseCommitSnapshotAndRollback(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	createWorkspace(t, app, nil, "build-workspace")
	headers := map[string]string{"X-Endge-Workspace": "build-workspace"}
	createQuery := func(identity string) {
		response := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": identity, "displayName": identity, "source": "query {}", "sourceVersion": 2}, headers)
		assertStatus(t, response, fiber.StatusCreated)
		response.Body.Close()
	}
	state := func() map[string]any {
		response := perform(t, app, http.MethodGet, "/api/v1/domain", nil, headers)
		assertStatus(t, response, fiber.StatusOK)
		return objectField(t, objectField(t, decodeObject(t, response), "workspace"), "state")
	}
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	_, _ = writer.Write([]byte(`{"format":"endge-bundle","version":1,"bundle":{"version":1,"programId":"test-program","compilerVersion":"program-v4","context":{},"catalog":{"folders":{},"documents":{}},"artifacts":{}}}`))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	metadata := func(state map[string]any) map[string]any {
		return map[string]any{"identity": "built-release", "displayName": "Built release", "description": "Комментарий к релизу", "workspaceId": state["id"], "generation": state["generation"], "headSequence": state["headSequence"], "commitMessage": "Build source", "buildMetadata": map[string]any{"version": 1, "programId": "test-program", "compilerVersion": "program-v4", "runtime": "ts-browser", "scope": "complete-model", "contextMode": "effective-context", "includeAst": false, "fileFormat": "gzip", "profile": map[string]any{"identity": "profile", "displayName": "Original profile", "revision": 3}}}
	}
	upload := func(value map[string]any, data []byte) *http.Response {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		encoded, _ := json.Marshal(value)
		if err := form.WriteField("metadata", string(encoded)); err != nil {
			t.Fatal(err)
		}
		file, err := form.CreateFormFile("bundle", "program.gz")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(data); err != nil {
			t.Fatal(err)
		}
		if err = form.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/releases/from-build", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("X-Endge-Workspace", "build-workspace")
		response, err := app.Test(request, 15000)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	createQuery("built-source")
	input := metadata(state())
	response := upload(input, zipped.Bytes())
	assertStatus(t, response, fiber.StatusCreated)
	release := decodeObject(t, response)
	if release["description"] != "Комментарий к релизу" {
		t.Fatalf("description lost: %#v", release)
	}
	build := objectField(t, release, "buildMetadata")
	if numberField(t, build, "sizeBytes") != zipped.Len() || len(stringField(t, build, "checksum")) != 64 {
		t.Fatalf("invalid binary metadata: %#v", build)
	}
	sourceID := stringField(t, release, "sourceCommitId")
	exported := perform(t, app, http.MethodGet, "/api/v1/releases/built-release/bundle", nil, headers)
	assertStatus(t, exported, fiber.StatusOK)
	body, _ := io.ReadAll(exported.Body)
	exported.Body.Close()
	if !bytes.Equal(body, zipped.Bytes()) {
		t.Fatal("stored Bundle changed")
	}
	snapshot := perform(t, app, http.MethodGet, "/api/v1/releases/built-release/export", nil, headers)
	assertStatus(t, snapshot, fiber.StatusOK)
	snapshot.Body.Close()
	list := perform(t, app, http.MethodGet, "/api/v1/releases", nil, headers)
	assertStatus(t, list, fiber.StatusOK)
	listBody, _ := io.ReadAll(list.Body)
	list.Body.Close()
	if bytes.Contains(listBody, []byte("compiled_bundle")) || bytes.Contains(listBody, []byte("\"data\"")) {
		t.Fatal("list loaded binary or snapshot")
	}
	// Existing source commit is reused and remains valid after later model changes.
	input["identity"] = "same-commit"
	input["sourceCommitId"] = sourceID
	reused := upload(input, zipped.Bytes())
	assertStatus(t, reused, fiber.StatusCreated)
	if stringField(t, decodeObject(t, reused), "sourceCommitId") != sourceID {
		t.Fatal("commit was not reused")
	}
	createQuery("later-source")
	next := metadata(state()) // duplicate release identity forces rollback after optional commit creation.
	var before, after int
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	conflict := upload(next, zipped.Bytes())
	assertStatus(t, conflict, fiber.StatusConflict)
	conflict.Body.Close()
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("failed release left an orphan commit")
	}
	next["identity"] = "mismatch"
	next["sourceCommitId"] = sourceID
	mismatch := upload(next, zipped.Bytes())
	assertStatus(t, mismatch, fiber.StatusConflict)
	mismatch.Body.Close()
	damaged := upload(next, zipped.Bytes()[:zipped.Len()-3])
	assertStatus(t, damaged, fiber.StatusBadRequest)
	damaged.Body.Close()
	// The original legacy endpoint still creates a snapshot-only release.
	legacy := perform(t, app, http.MethodPost, "/api/v1/releases", map[string]any{"identity": "legacy", "displayName": "Legacy", "sourceCommitId": sourceID}, headers)
	assertStatus(t, legacy, fiber.StatusCreated)
	if value := decodeObject(t, legacy)["buildMetadata"]; value != nil {
		t.Fatalf("legacy build metadata: %#v", value)
	}
	absent := perform(t, app, http.MethodGet, "/api/v1/releases/legacy/bundle", nil, headers)
	assertStatus(t, absent, fiber.StatusNotFound)
	absent.Body.Close()
}
