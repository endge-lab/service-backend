//go:build e2e

package e2e_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
	"github.com/gofiber/fiber/v2"
)

// TestCommitReleaseBackupAndImportFlow проверяет переносимый snapshot и точки восстановления целиком через HTTP.
func TestCommitReleaseBackupAndImportFlow(t *testing.T) {
	database := postgresSuite.NewDatabase(t)
	app := support.NewTestApp(t, database, support.DevConfig())
	headers := map[string]string{"X-Endge-Workspace": "default"}

	created := perform(t, app, http.MethodPost, "/api/v1/queries", map[string]any{"identity": "portable-query", "displayName": "Portable", "source": "query {}", "sourceVersion": 2}, headers)
	assertStatus(t, created, fiber.StatusCreated)
	created.Body.Close()
	patchHeaders := cloneHeaders(headers)
	patchHeaders["If-Match"] = `"1"`
	changed := perform(t, app, http.MethodPatch, "/api/v1/queries/portable-query", map[string]any{"source": "query { changed }"}, patchHeaders)
	assertStatus(t, changed, fiber.StatusOK)
	changed.Body.Close()
	revisionList := perform(t, app, http.MethodGet, "/api/v1/domain/documents/queries/portable-query/revisions", nil, headers)
	assertStatus(t, revisionList, fiber.StatusOK)
	revisionBody := decodeObject(t, revisionList)
	items, ok := revisionBody["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("до restore ожидались две revision: %#v", revisionBody)
	}
	firstRevisionID := ""
	for _, item := range items {
		revision, _ := item.(map[string]any)
		if numberField(t, revision, "revisionNumber") == 1 {
			firstRevisionID = stringField(t, revision, "id")
		}
	}
	if firstRevisionID == "" {
		t.Fatal("первая revision не найдена")
	}
	restoreHeaders := cloneHeaders(headers)
	restoreHeaders["If-Match"] = `"2"`
	restoredRevision := perform(t, app, http.MethodPost, "/api/v1/domain/documents/queries/portable-query/revisions/"+firstRevisionID+"/restore", nil, restoreHeaders)
	assertStatus(t, restoredRevision, fiber.StatusOK)
	if restoredRevision.Header.Get("ETag") != `"3"` {
		t.Fatalf("revision restore ETag=%q", restoredRevision.Header.Get("ETag"))
	}
	restoredBody := decodeObject(t, restoredRevision)
	if stringField(t, restoredBody, "source") != "query {}" {
		t.Fatalf("revision restore не вернул исходный snapshot: %#v", restoredBody)
	}
	live := perform(t, app, http.MethodGet, "/api/v1/domain", nil, headers)
	assertStatus(t, live, fiber.StatusOK)
	liveBody := decodeObject(t, live)
	state := objectField(t, objectField(t, liveBody, "workspace"), "state")
	headSequence := int64(numberField(t, state, "headSequence"))

	plan := perform(t, app, http.MethodPost, "/api/v1/commits/plan", nil, headers)
	assertStatus(t, plan, fiber.StatusOK)
	if numberField(t, decodeObject(t, plan), "revisionCount") < 1 {
		t.Fatal("commit plan не содержит pending revisions")
	}
	commit := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "Portable baseline", "revisionPolicy": "preserve", "expectedHeadSequence": headSequence}, headers)
	assertStatus(t, commit, fiber.StatusCreated)
	commitID := stringField(t, decodeObject(t, commit), "id")
	nothing := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "Empty", "revisionPolicy": "preserve", "expectedHeadSequence": headSequence}, headers)
	assertStatus(t, nothing, fiber.StatusConflict)

	release := perform(t, app, http.MethodPost, "/api/v1/releases", map[string]any{"identity": "portable-release", "displayName": "Portable Release", "sourceCommitId": commitID}, headers)
	assertStatus(t, release, fiber.StatusCreated)
	releaseExport := perform(t, app, http.MethodGet, "/api/v1/releases/last/export", nil, headers)
	assertStatus(t, releaseExport, fiber.StatusOK)
	releaseBundle := decodeObject(t, releaseExport)
	assertPortableBundle(t, releaseBundle)

	afterReleaseHeaders := cloneHeaders(headers)
	afterReleaseHeaders["If-Match"] = `"3"`
	afterRelease := perform(t, app, http.MethodPatch, "/api/v1/queries/portable-query", map[string]any{"source": "query { afterRelease }"}, afterReleaseHeaders)
	assertStatus(t, afterRelease, fiber.StatusOK)
	afterRelease.Body.Close()
	secondHead := currentHeadSequence(t, app, headers)
	secondCommitResponse := perform(t, app, http.MethodPost, "/api/v1/commits", map[string]any{"message": "Second state", "revisionPolicy": "preserve", "expectedHeadSequence": secondHead}, headers)
	assertStatus(t, secondCommitResponse, fiber.StatusCreated)
	secondCommitID := stringField(t, decodeObject(t, secondCommitResponse), "id")

	releaseRestorePlan := perform(t, app, http.MethodPost, "/api/v1/releases/portable-release/restore/plan", nil, headers)
	assertStatus(t, releaseRestorePlan, fiber.StatusOK)
	releaseRestorePlan.Body.Close()
	releaseRestore := perform(t, app, http.MethodPost, "/api/v1/releases/portable-release/restore", map[string]any{"expectedHeadSequence": currentHeadSequence(t, app, headers)}, headers)
	assertStatus(t, releaseRestore, fiber.StatusCreated)
	releaseRestore.Body.Close()
	restoredFromRelease := perform(t, app, http.MethodGet, "/api/v1/queries/portable-query", nil, headers)
	assertStatus(t, restoredFromRelease, fiber.StatusOK)
	if stringField(t, decodeObject(t, restoredFromRelease), "source") != "query {}" {
		t.Fatal("release restore не вернул сохранённое состояние")
	}

	commitRestorePlan := perform(t, app, http.MethodPost, "/api/v1/commits/"+secondCommitID+"/restore/plan", nil, headers)
	assertStatus(t, commitRestorePlan, fiber.StatusOK)
	commitRestorePlan.Body.Close()
	commitRestore := perform(t, app, http.MethodPost, "/api/v1/commits/"+secondCommitID+"/restore", map[string]any{"expectedHeadSequence": currentHeadSequence(t, app, headers)}, headers)
	assertStatus(t, commitRestore, fiber.StatusCreated)
	commitRestore.Body.Close()
	restoredFromCommit := perform(t, app, http.MethodGet, "/api/v1/queries/portable-query", nil, headers)
	assertStatus(t, restoredFromCommit, fiber.StatusOK)
	if stringField(t, decodeObject(t, restoredFromCommit), "source") != "query { afterRelease }" {
		t.Fatal("commit restore не вернул точное состояние второго commit")
	}

	manualBackup := perform(t, app, http.MethodPost, "/api/v1/domain/backups", map[string]any{"description": "Перед импортом"}, headers)
	assertStatus(t, manualBackup, fiber.StatusCreated)
	backupID := stringField(t, decodeObject(t, manualBackup), "id")
	backupLast := perform(t, app, http.MethodGet, "/api/v1/domain/backups/last", nil, headers)
	assertStatus(t, backupLast, fiber.StatusOK)
	if stringField(t, decodeObject(t, backupLast), "id") != backupID {
		t.Fatal("alias last не вернул последний backup")
	}
	backupExport := perform(t, app, http.MethodGet, "/api/v1/domain/backups/last/export", nil, headers)
	assertStatus(t, backupExport, fiber.StatusOK)
	assertPortableBundle(t, decodeObject(t, backupExport))
	archive := perform(t, app, http.MethodGet, "/api/v1/domain/backups/archive", nil, headers)
	assertStatus(t, archive, fiber.StatusOK)
	assertBackupArchive(t, archive)

	domainExport := perform(t, app, http.MethodGet, "/api/v1/domain/export", nil, headers)
	assertStatus(t, domainExport, fiber.StatusOK)
	snapshot := decodeObject(t, domainExport)
	assertPortableBundle(t, snapshot)
	importPlan := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", map[string]any{"snapshot": snapshot}, headers)
	assertStatus(t, importPlan, fiber.StatusOK)
	planBody := decodeObject(t, importPlan)
	if valid, _ := planBody["valid"].(bool); !valid {
		t.Fatalf("валидный export не прошёл import plan: %#v", planBody)
	}
	importHeaders := cloneHeaders(headers)
	importHeaders["If-Match"] = stringField(t, planBody, "targetETag")
	wrongConfirmation := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{"planId": stringField(t, planBody, "planId"), "confirmation": "wrong-workspace"}, importHeaders)
	assertStatus(t, wrongConfirmation, fiber.StatusBadRequest)
	staleImportHeaders := cloneHeaders(headers)
	staleImportHeaders["If-Match"] = `"stale-generation:0"`
	staleImport := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{"planId": stringField(t, planBody, "planId"), "confirmation": "default"}, staleImportHeaders)
	assertStatus(t, staleImport, fiber.StatusPreconditionFailed)

	var backupsBeforeFailure, commitsBeforeFailure int
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_snapshot_backups`).Scan(&backupsBeforeFailure); err != nil {
		t.Fatalf("посчитать backups перед rollback-проверкой: %v", err)
	}
	if err := database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits`).Scan(&commitsBeforeFailure); err != nil {
		t.Fatalf("посчитать commits перед rollback-проверкой: %v", err)
	}
	_, err := database.Pool.Exec(t.Context(), `
		CREATE FUNCTION endge_test_fail_import() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced import failure'; END $$;
		CREATE TRIGGER endge_test_fail_import_trigger BEFORE INSERT ON queries
		FOR EACH ROW EXECUTE FUNCTION endge_test_fail_import()`)
	if err != nil {
		t.Fatalf("установить test-only import failpoint: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Pool.Exec(t.Context(), `DROP TRIGGER IF EXISTS endge_test_fail_import_trigger ON queries; DROP FUNCTION IF EXISTS endge_test_fail_import()`)
	})
	failedImport := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{"planId": stringField(t, planBody, "planId"), "confirmation": "default"}, importHeaders)
	assertStatus(t, failedImport, fiber.StatusInternalServerError)
	failedBody := decodeObject(t, failedImport)
	if bytes.Contains([]byte(stringField(t, failedBody, "message")), []byte("forced import failure")) {
		t.Fatalf("HTTP envelope раскрыл внутреннюю PostgreSQL ошибку: %#v", failedBody)
	}
	if _, err = database.Pool.Exec(t.Context(), `DROP TRIGGER endge_test_fail_import_trigger ON queries; DROP FUNCTION endge_test_fail_import()`); err != nil {
		t.Fatalf("удалить test-only import failpoint: %v", err)
	}
	var backupsAfterFailure, commitsAfterFailure int
	if err = database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_snapshot_backups`).Scan(&backupsAfterFailure); err != nil {
		t.Fatalf("посчитать backups после rollback: %v", err)
	}
	if err = database.Pool.QueryRow(t.Context(), `SELECT count(*) FROM workspace_commits`).Scan(&commitsAfterFailure); err != nil {
		t.Fatalf("посчитать commits после rollback: %v", err)
	}
	if backupsAfterFailure != backupsBeforeFailure || commitsAfterFailure != commitsBeforeFailure {
		t.Fatalf("ошибочный import оставил partial state: backups %d->%d, commits %d->%d", backupsBeforeFailure, backupsAfterFailure, commitsBeforeFailure, commitsAfterFailure)
	}
	stillPresent := perform(t, app, http.MethodGet, "/api/v1/queries/portable-query", nil, headers)
	assertStatus(t, stillPresent, fiber.StatusOK)
	stillPresent.Body.Close()

	freshPlan := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", map[string]any{"snapshot": snapshot}, headers)
	assertStatus(t, freshPlan, fiber.StatusOK)
	freshPlanBody := decodeObject(t, freshPlan)
	importHeaders["If-Match"] = stringField(t, freshPlanBody, "targetETag")
	importResult := perform(t, app, http.MethodPost, "/api/v1/domain/import", map[string]any{"planId": stringField(t, freshPlanBody, "planId"), "confirmation": "default"}, importHeaders)
	assertStatus(t, importResult, fiber.StatusCreated)
	resultBody := decodeObject(t, importResult)
	if stringField(t, objectField(t, resultBody, "backup"), "kind") != "pre_import" || stringField(t, resultBody, "initialCommitId") == "" {
		t.Fatalf("import не создал backup/initial commit: %#v", resultBody)
	}
	importedQuery := perform(t, app, http.MethodGet, "/api/v1/queries/portable-query", nil, headers)
	assertStatus(t, importedQuery, fiber.StatusOK)

	secretSnapshot := cloneJSON(snapshot)
	workspace := objectField(t, secretSnapshot, "workspace")
	workspace["configuration"] = map[string]any{"nested": map[string]any{"password": "forbidden"}}
	secretPlan := perform(t, app, http.MethodPost, "/api/v1/domain/import/plan", map[string]any{"snapshot": secretSnapshot}, headers)
	assertStatus(t, secretPlan, fiber.StatusOK)
	secretPlanBody := decodeObject(t, secretPlan)
	if valid, _ := secretPlanBody["valid"].(bool); valid {
		t.Fatalf("import plan принял secret: %#v", secretPlanBody)
	}

	backupList := perform(t, app, http.MethodGet, "/api/v1/domain/backups?limit=100&offset=0", nil, headers)
	assertStatus(t, backupList, fiber.StatusOK)
	if numberField(t, decodeObject(t, backupList), "total") < 2 {
		t.Fatal("после import не сохранены manual и pre_import backups")
	}
}

func currentHeadSequence(t *testing.T, app *fiber.App, headers map[string]string) int64 {
	t.Helper()
	response := perform(t, app, http.MethodGet, "/api/v1/domain", nil, headers)
	assertStatus(t, response, fiber.StatusOK)
	state := objectField(t, objectField(t, decodeObject(t, response), "workspace"), "state")
	return int64(numberField(t, state, "headSequence"))
}

func assertPortableBundle(t *testing.T, value map[string]any) {
	t.Helper()
	if stringField(t, value, "kind") != "workspace-snapshot" || numberField(t, value, "schemaVersion") != 1 {
		t.Fatalf("неверный portable bundle header: %#v", value)
	}
	if _, ok := value["documents"].(map[string]any); !ok {
		t.Fatalf("documents отсутствуют в portable bundle: %#v", value)
	}
	raw, _ := json.Marshal(value)
	for _, forbidden := range []string{`"memberships"`, `"revisions"`, `"commits"`, `"releases"`, `"password"`, `"clientSecret"`, `"accessToken"`} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("portable bundle содержит запрещённое поле %s", forbidden)
		}
	}
}

func assertBackupArchive(t *testing.T, response *http.Response) {
	t.Helper()
	raw, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatalf("прочитать ZIP: %v", err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("открыть ZIP: %v", err)
	}
	manifest, snapshots := false, 0
	for _, file := range archive.File {
		if file.Name == "manifest.json" {
			manifest = true
		}
		if len(file.Name) > len("backups/") && file.Name[:len("backups/")] == "backups/" {
			snapshots++
		}
	}
	if !manifest || snapshots == 0 {
		t.Fatalf("ZIP не содержит manifest/snapshots: files=%d", len(archive.File))
	}
}

func cloneJSON(value map[string]any) map[string]any {
	raw, _ := json.Marshal(value)
	result := map[string]any{}
	_ = json.Unmarshal(raw, &result)
	return result
}
